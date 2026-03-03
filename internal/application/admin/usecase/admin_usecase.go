package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric/masterfabric_go_basic/internal/application/admin/dto"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/model"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/policy"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/repository"
	domainErr "github.com/masterfabric/masterfabric_go_basic/internal/shared/errors"
)

// ListUsersUseCase returns a paginated list of all users (admin only).
type ListUsersUseCase struct {
	userRepo repository.UserRepository
}

func NewListUsersUseCase(userRepo repository.UserRepository) *ListUsersUseCase {
	return &ListUsersUseCase{userRepo: userRepo}
}

func (uc *ListUsersUseCase) Execute(ctx context.Context, page, pageSize int) (*dto.AdminUserListResponse, error) {
	if err := policy.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	users, err := uc.userRepo.List(ctx, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("listUsers: %w", err)
	}
	total, err := uc.userRepo.CountAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listUsers count: %w", err)
	}

	result := make([]*dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, userToAdminDTO(u))
	}
	return &dto.AdminUserListResponse{
		Users:      result,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// GetUserByIDUseCase fetches a single user by ID (admin only).
type GetUserByIDUseCase struct {
	userRepo repository.UserRepository
}

func NewGetUserByIDUseCase(userRepo repository.UserRepository) *GetUserByIDUseCase {
	return &GetUserByIDUseCase{userRepo: userRepo}
}

func (uc *GetUserByIDUseCase) Execute(ctx context.Context, targetID string) (*dto.AdminUserResponse, error) {
	if err := policy.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(targetID)
	if err != nil {
		return nil, domainErr.ErrUserNotFound
	}
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return userToAdminDTO(user), nil
}

// SuspendUserUseCase sets a user's status to suspended or active (admin only).
type SuspendUserUseCase struct {
	userRepo repository.UserRepository
}

func NewSuspendUserUseCase(userRepo repository.UserRepository) *SuspendUserUseCase {
	return &SuspendUserUseCase{userRepo: userRepo}
}

func (uc *SuspendUserUseCase) Execute(ctx context.Context, req *dto.SuspendUserRequest) (*dto.AdminUserResponse, error) {
	if err := policy.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		return nil, domainErr.ErrUserNotFound
	}
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Suspend {
		user.Status = model.UserStatusSuspended
	} else {
		user.Status = model.UserStatusActive
	}
	user.UpdatedAt = time.Now().UTC()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("suspendUser: %w", err)
	}
	return userToAdminDTO(user), nil
}

// ChangeUserRoleUseCase updates a user's role (admin only).
type ChangeUserRoleUseCase struct {
	userRepo repository.UserRepository
}

func NewChangeUserRoleUseCase(userRepo repository.UserRepository) *ChangeUserRoleUseCase {
	return &ChangeUserRoleUseCase{userRepo: userRepo}
}

func (uc *ChangeUserRoleUseCase) Execute(ctx context.Context, req *dto.ChangeRoleRequest) (*dto.AdminUserResponse, error) {
	if err := policy.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		return nil, domainErr.ErrUserNotFound
	}
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Role = model.UserRole(req.Role)
	user.UpdatedAt = time.Now().UTC()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("changeRole: %w", err)
	}
	return userToAdminDTO(user), nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func userToAdminDTO(u *model.User) *dto.AdminUserResponse {
	return &dto.AdminUserResponse{
		ID:             u.ID.String(),
		Email:          u.Email,
		DisplayName:    u.DisplayName,
		AvatarURL:      u.AvatarURL,
		Bio:            u.Bio,
		Status:         string(u.Status),
		Role:           string(u.Role),
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      u.UpdatedAt.Format(time.RFC3339),
		PhoneNumber:    u.PhoneNumber,
		DateOfBirth:    u.DateOfBirth,
		Gender:         string(u.Gender),
		Location:       u.Location,
		WebsiteURL:     u.WebsiteURL,
		SocialTwitter:  u.SocialTwitter,
		SocialGitHub:   u.SocialGitHub,
		SocialLinkedIn: u.SocialLinkedIn,
		Language:       u.Language,
	}
}
