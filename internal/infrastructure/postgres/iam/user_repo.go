package iam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/model"
	domainErr "github.com/masterfabric/masterfabric_go_basic/internal/shared/errors"
)

// UserRepo is the PostgreSQL implementation of domain UserRepository.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

const (
	sqlCreateUser = `
		INSERT INTO users (id, email, password_hash, display_name, avatar_url, bio, status, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	sqlFindUserByID = `
		SELECT id, email, password_hash, display_name, avatar_url, bio, status, role, created_at, updated_at,
		       phone_number, date_of_birth, gender, location, website_url,
		       social_twitter, social_github, social_linkedin, language
		FROM users WHERE id = $1`

	sqlFindUserByEmail = `
		SELECT id, email, password_hash, display_name, avatar_url, bio, status, role, created_at, updated_at,
		       phone_number, date_of_birth, gender, location, website_url,
		       social_twitter, social_github, social_linkedin, language
		FROM users WHERE email = $1`

	sqlUpdateUser = `
		UPDATE users
		SET email = $2, password_hash = $3, display_name = $4, avatar_url = $5, bio = $6,
		    status = $7, role = $8, updated_at = $9,
		    phone_number = $10, date_of_birth = $11, gender = $12, location = $13,
		    website_url = $14, social_twitter = $15, social_github = $16,
		    social_linkedin = $17, language = $18
		WHERE id = $1`

	sqlDeleteUser = `DELETE FROM users WHERE id = $1`

	sqlListUsers = `
		SELECT id, email, password_hash, display_name, avatar_url, bio, status, role, created_at, updated_at,
		       phone_number, date_of_birth, gender, location, website_url,
		       social_twitter, social_github, social_linkedin, language
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	sqlCountUsers = `SELECT COUNT(*) FROM users`

	// ── Address SQL ────────────────────────────────────────────────────────────

	sqlCreateAddress = `
		INSERT INTO user_addresses
		    (id, user_id, title, address_line1, address_line2, city, state, postal_code, country, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	// UpsertAddress: insert or update by (user_id, id).
	// When is_default becomes true we first clear any existing default for that user,
	// then insert/update the target row. That logic is handled in Go to avoid a CTE
	// that would need two round-trips anyway.
	sqlInsertOrUpdateAddress = `
		INSERT INTO user_addresses
		    (id, user_id, title, address_line1, address_line2, city, state, postal_code, country, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE
		    SET title         = EXCLUDED.title,
		        address_line1 = EXCLUDED.address_line1,
		        address_line2 = EXCLUDED.address_line2,
		        city          = EXCLUDED.city,
		        state         = EXCLUDED.state,
		        postal_code   = EXCLUDED.postal_code,
		        country       = EXCLUDED.country,
		        is_default    = EXCLUDED.is_default,
		        updated_at    = EXCLUDED.updated_at
		RETURNING id, user_id, title, address_line1, address_line2, city, state, postal_code, country, is_default, created_at, updated_at`

	sqlClearDefaultAddress = `
		UPDATE user_addresses SET is_default = FALSE
		WHERE user_id = $1 AND is_default = TRUE AND id != $2`

	sqlFindAddressesByUserID = `
		SELECT id, user_id, title, address_line1, address_line2, city, state, postal_code, country, is_default, created_at, updated_at
		FROM user_addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at ASC`

	sqlFindDefaultAddressByUserID = `
		SELECT id, user_id, title, address_line1, address_line2, city, state, postal_code, country, is_default, created_at, updated_at
		FROM user_addresses
		WHERE user_id = $1 AND is_default = TRUE
		LIMIT 1`
)

// Create persists a new user record.
func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	_, err := r.db.Exec(ctx, sqlCreateUser,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.AvatarURL,
		user.Bio,
		string(user.Status),
		string(user.Role),
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

// FindByID retrieves a user by primary key.
func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.db.QueryRow(ctx, sqlFindUserByID, id)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	return user, nil
}

// FindByEmail retrieves a user by email address.
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.db.QueryRow(ctx, sqlFindUserByEmail, email)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	return user, nil
}

// Update persists changes to an existing user.
func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()
	tag, err := r.db.Exec(ctx, sqlUpdateUser,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.AvatarURL,
		user.Bio,
		string(user.Status),
		string(user.Role),
		user.UpdatedAt,
		user.PhoneNumber,
		user.DateOfBirth,
		string(user.Gender),
		user.Location,
		user.WebsiteURL,
		user.SocialTwitter,
		user.SocialGitHub,
		user.SocialLinkedIn,
		user.Language,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.ErrUserNotFound
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, sqlDeleteUser, id)
	if err != nil {
		return fmt.Errorf("userRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.ErrUserNotFound
	}
	return nil
}

// List returns a paginated list of all users ordered by creation date descending.
func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	rows, err := r.db.Query(ctx, sqlListUsers, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("userRepo.List: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("userRepo.List scan: %w", err)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// CountAll returns the total number of users.
func (r *UserRepo) CountAll(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, sqlCountUsers).Scan(&count); err != nil {
		return 0, fmt.Errorf("userRepo.CountAll: %w", err)
	}
	return count, nil
}

// scanUser reads a single pgx row into a User entity.
func scanUser(row pgx.Row) (*model.User, error) {
	var (
		u      model.User
		status string
		role   string
		gender string
	)
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.AvatarURL,
		&u.Bio,
		&status,
		&role,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.PhoneNumber,
		&u.DateOfBirth,
		&gender,
		&u.Location,
		&u.WebsiteURL,
		&u.SocialTwitter,
		&u.SocialGitHub,
		&u.SocialLinkedIn,
		&u.Language,
	)
	if err != nil {
		return nil, err
	}
	u.Status = model.UserStatus(status)
	u.Role = model.UserRole(role)
	u.Gender = model.UserGender(gender)
	return &u, nil
}

// ── Address methods ────────────────────────────────────────────────────────────

// CreateAddress inserts a brand-new address row.
func (r *UserRepo) CreateAddress(ctx context.Context, addr *model.UserAddress) error {
	now := time.Now().UTC()
	addr.CreatedAt = now
	addr.UpdatedAt = now

	if addr.IsDefault {
		// Clear any existing default for this user before inserting
		if _, err := r.db.Exec(ctx, sqlClearDefaultAddress, addr.UserID, addr.ID); err != nil {
			return fmt.Errorf("userRepo.CreateAddress clear default: %w", err)
		}
	}

	_, err := r.db.Exec(ctx, sqlCreateAddress,
		addr.ID, addr.UserID,
		addr.Title, addr.AddressLine1, addr.AddressLine2,
		addr.City, addr.State, addr.PostalCode, addr.Country,
		addr.IsDefault, addr.CreatedAt, addr.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepo.CreateAddress: %w", err)
	}
	return nil
}

// UpsertAddress inserts or updates an address by its ID and returns the persisted row.
func (r *UserRepo) UpsertAddress(ctx context.Context, addr *model.UserAddress) (*model.UserAddress, error) {
	addr.UpdatedAt = time.Now().UTC()
	if addr.CreatedAt.IsZero() {
		addr.CreatedAt = addr.UpdatedAt
	}

	if addr.IsDefault {
		// Clear any existing default for this user before the upsert
		if _, err := r.db.Exec(ctx, sqlClearDefaultAddress, addr.UserID, addr.ID); err != nil {
			return nil, fmt.Errorf("userRepo.UpsertAddress clear default: %w", err)
		}
	}

	row := r.db.QueryRow(ctx, sqlInsertOrUpdateAddress,
		addr.ID, addr.UserID,
		addr.Title, addr.AddressLine1, addr.AddressLine2,
		addr.City, addr.State, addr.PostalCode, addr.Country,
		addr.IsDefault, addr.CreatedAt, addr.UpdatedAt,
	)
	return scanAddress(row)
}

// FindAddressByUserID returns all addresses for a user (default first, then by creation date).
func (r *UserRepo) FindAddressByUserID(ctx context.Context, userID uuid.UUID) ([]*model.UserAddress, error) {
	rows, err := r.db.Query(ctx, sqlFindAddressesByUserID, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindAddressByUserID: %w", err)
	}
	defer rows.Close()

	var addrs []*model.UserAddress
	for rows.Next() {
		addr, err := scanAddress(rows)
		if err != nil {
			return nil, fmt.Errorf("userRepo.FindAddressByUserID scan: %w", err)
		}
		addrs = append(addrs, addr)
	}
	return addrs, rows.Err()
}

// FindDefaultAddressByUserID returns the single default address for a user, or nil if none set.
func (r *UserRepo) FindDefaultAddressByUserID(ctx context.Context, userID uuid.UUID) (*model.UserAddress, error) {
	row := r.db.QueryRow(ctx, sqlFindDefaultAddressByUserID, userID)
	addr, err := scanAddress(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no default address is not an error
		}
		return nil, fmt.Errorf("userRepo.FindDefaultAddressByUserID: %w", err)
	}
	return addr, nil
}

// scanAddress reads a single pgx row into a UserAddress entity.
func scanAddress(row pgx.Row) (*model.UserAddress, error) {
	var a model.UserAddress
	err := row.Scan(
		&a.ID, &a.UserID,
		&a.Title, &a.AddressLine1, &a.AddressLine2,
		&a.City, &a.State, &a.PostalCode, &a.Country,
		&a.IsDefault, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
