package resolver

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	adminUC "github.com/masterfabric/masterfabric_go_basic/internal/application/admin/usecase"
	authUC "github.com/masterfabric/masterfabric_go_basic/internal/application/auth/usecase"
	settingsUC "github.com/masterfabric/masterfabric_go_basic/internal/application/settings/usecase"
	userUC "github.com/masterfabric/masterfabric_go_basic/internal/application/user/usecase"
)

// Resolver is the root dependency container wired up in main.
type Resolver struct {
	// Auth
	RegisterUC *authUC.RegisterUseCase
	LoginUC    *authUC.LoginUseCase
	RefreshUC  *authUC.RefreshUseCase
	LogoutUC   *authUC.LogoutUseCase
	// User
	GetProfileUC        *userUC.GetProfileUseCase
	UpdateProfileUC     *userUC.UpdateProfileUseCase
	DeleteAccountUC     *userUC.DeleteAccountUseCase
	GetAddressUC        *userUC.GetAddressUseCase
	GetDefaultAddressUC *userUC.GetDefaultAddressUseCase
	UpsertAddressUC     *userUC.UpsertAddressUseCase
	// Settings
	GetUserSettingsUC    *settingsUC.GetUserSettingsUseCase
	UpdateUserSettingsUC *settingsUC.UpdateUserSettingsUseCase
	GetAppSettingsUC     *settingsUC.GetAppSettingsUseCase
	// Admin
	ListUsersUC   *adminUC.ListUsersUseCase
	GetUserByIDUC *adminUC.GetUserByIDUseCase
	SuspendUserUC *adminUC.SuspendUserUseCase
	ChangeRoleUC  *adminUC.ChangeUserRoleUseCase
}
