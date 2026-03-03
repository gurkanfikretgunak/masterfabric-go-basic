package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/vektah/gqlparser/v2/ast"

	adminUC "github.com/masterfabric/masterfabric_go_basic/internal/application/admin/usecase"
	authUC "github.com/masterfabric/masterfabric_go_basic/internal/application/auth/usecase"
	settingsUC "github.com/masterfabric/masterfabric_go_basic/internal/application/settings/usecase"
	userUC "github.com/masterfabric/masterfabric_go_basic/internal/application/user/usecase"
	infraAuth "github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/auth"
	"github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/graphql/generated"
	"github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/graphql/resolver"
	iamPG "github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/postgres/iam"
	"github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/postgres/migrations"
	settingsPG "github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/postgres/settings"
	"github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/rabbitmq"
	infraRedis "github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/redis"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/cache"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/config"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/database"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/events"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/health"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/logger"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/middleware"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/version"
)

func main() {
	// ── Config ──────────────────────────────────────────────────────────────
	cfg := config.Load()
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("starting masterfabric_go_basic", slog.String("version", version.Version))

	// ── Database ────────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	// ── Migrations ──────────────────────────────────────────────────────────
	if err := migrations.Run(cfg.Database.DSN, log); err != nil {
		log.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	// ── Redis ───────────────────────────────────────────────────────────────
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		log.Warn("redis unavailable — tokens will not be blacklisted", slog.Any("error", err))
	}
	cacheHandler := infraRedis.NewCacheHandler(redisClient)

	// ── Event bus ───────────────────────────────────────────────────────────
	var eventBus events.EventBus
	if cfg.RabbitMQ.Enabled {
		bus, err := rabbitmq.NewBus(cfg.RabbitMQ.URL, cfg.RabbitMQ.Exchange, log)
		if err != nil {
			log.Warn("rabbitmq unavailable — using in-process bus", slog.Any("error", err))
			eventBus = rabbitmq.NewInProcessBus(log)
		} else {
			eventBus = bus
			defer bus.Close()
		}
	} else {
		log.Info("rabbitmq disabled — using in-process bus")
		eventBus = rabbitmq.NewInProcessBus(log)
	}

	// ── Infrastructure ──────────────────────────────────────────────────────
	userRepo := iamPG.NewUserRepo(pool)
	userSettingsRepo := settingsPG.NewUserSettingsRepo(pool)
	appSettingsRepo := settingsPG.NewAppSettingsRepo(pool)

	jwtSvc := infraAuth.NewJWTService(cfg.JWT, cacheHandler)

	// ── Resolver (DI root) ──────────────────────────────────────────────────
	res := &resolver.Resolver{
		RegisterUC: authUC.NewRegisterUseCase(userRepo, jwtSvc, eventBus),
		LoginUC:    authUC.NewLoginUseCase(userRepo, jwtSvc, eventBus),
		RefreshUC:  authUC.NewRefreshUseCase(userRepo, jwtSvc),
		LogoutUC:   authUC.NewLogoutUseCase(jwtSvc),

		GetProfileUC:        userUC.NewGetProfileUseCase(userRepo),
		UpdateProfileUC:     userUC.NewUpdateProfileUseCase(userRepo, eventBus),
		DeleteAccountUC:     userUC.NewDeleteAccountUseCase(userRepo, eventBus),
		GetAddressUC:        userUC.NewGetAddressUseCase(userRepo),
		GetDefaultAddressUC: userUC.NewGetDefaultAddressUseCase(userRepo),
		UpsertAddressUC:     userUC.NewUpsertAddressUseCase(userRepo),

		GetUserSettingsUC:    settingsUC.NewGetUserSettingsUseCase(userSettingsRepo),
		UpdateUserSettingsUC: settingsUC.NewUpdateUserSettingsUseCase(userSettingsRepo, eventBus),
		GetAppSettingsUC:     settingsUC.NewGetAppSettingsUseCase(appSettingsRepo),

		ListUsersUC:   adminUC.NewListUsersUseCase(userRepo),
		GetUserByIDUC: adminUC.NewGetUserByIDUseCase(userRepo),
		SuspendUserUC: adminUC.NewSuspendUserUseCase(userRepo),
		ChangeRoleUC:  adminUC.NewChangeUserRoleUseCase(userRepo),
	}

	// ── GraphQL server ──────────────────────────────────────────────────────
	// Build the server manually so we can conditionally enable introspection.
	gqlSrv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: res}))
	gqlSrv.AddTransport(transport.Websocket{KeepAlivePingInterval: 10 * time.Second})
	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.AddTransport(transport.MultipartForm{})
	gqlSrv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	if cfg.GraphQL.Introspection {
		gqlSrv.Use(extension.Introspection{})
	} else {
		log.Info("GraphQL introspection disabled")
	}
	gqlSrv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})

	// ── HTTP router ─────────────────────────────────────────────────────────
	router := chi.NewRouter()
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.RealIP)
	router.Use(chiMiddleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	router.Use(middleware.RequestIDMiddleware)
	router.Use(middleware.AuthMiddleware(jwtSvc))

	// GraphQL endpoint
	router.Handle("/graphql", gqlSrv)

	// Playground (disable behind an env flag in production if desired)
	router.Handle("/", playground.Handler("MasterFabric GraphQL", "/graphql"))

	// Health checks
	// GET /health        — liveness  (always 200 if process is up)
	// GET /health/ready  — readiness (503 if postgres or redis is down)
	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, version.Version)
	})
	router.Get("/health/ready", health.Handler(pool, redisClient))

	// ── HTTP server ─────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server listening", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down server...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("graceful shutdown failed", slog.Any("error", err))
	}
	log.Info("server stopped")
}
