package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric/masterfabric_go_basic/internal/application/user/dto"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/event"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/model"
	"github.com/masterfabric/masterfabric_go_basic/internal/domain/iam/repository"
	domainErr "github.com/masterfabric/masterfabric_go_basic/internal/shared/errors"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/events"
)

// GetProfileUseCase returns the user profile for the given user ID.
type GetProfileUseCase struct {
	userRepo repository.UserRepository
}

// NewGetProfileUseCase constructs a GetProfileUseCase.
func NewGetProfileUseCase(userRepo repository.UserRepository) *GetProfileUseCase {
	return &GetProfileUseCase{userRepo: userRepo}
}

// Execute fetches the profile.
func (uc *GetProfileUseCase) Execute(ctx context.Context, userID string) (*dto.UserProfileResponse, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, domainErr.ErrUserNotFound
	}

	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil || user == nil {
		return nil, domainErr.ErrUserNotFound
	}

	return userToProfileResponse(user), nil
}

// UpdateProfileUseCase modifies mutable user fields.
type UpdateProfileUseCase struct {
	userRepo repository.UserRepository
	eventBus events.EventBus
}

// NewUpdateProfileUseCase constructs an UpdateProfileUseCase.
func NewUpdateProfileUseCase(userRepo repository.UserRepository, eventBus events.EventBus) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{userRepo: userRepo, eventBus: eventBus}
}

// Execute applies partial updates to a user profile.
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, req *dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	id, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, domainErr.ErrUserNotFound
	}

	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil || user == nil {
		return nil, domainErr.ErrUserNotFound
	}

	// Apply partial updates — only overwrite fields that were provided
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}
	if req.PhoneNumber != nil {
		user.PhoneNumber = *req.PhoneNumber
	}
	if req.DateOfBirth != nil {
		user.DateOfBirth = req.DateOfBirth
	}
	if req.Gender != nil {
		user.Gender = model.UserGender(*req.Gender)
	}
	if req.Location != nil {
		user.Location = *req.Location
	}
	if req.WebsiteURL != nil {
		user.WebsiteURL = *req.WebsiteURL
	}
	if req.SocialTwitter != nil {
		user.SocialTwitter = *req.SocialTwitter
	}
	if req.SocialGitHub != nil {
		user.SocialGitHub = *req.SocialGitHub
	}
	if req.SocialLinkedIn != nil {
		user.SocialLinkedIn = *req.SocialLinkedIn
	}
	if req.Language != nil {
		user.Language = *req.Language
	}
	user.UpdatedAt = time.Now().UTC()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	_ = uc.eventBus.Publish(ctx, events.TopicUserUpdated, events.Event{
		Type:    event.EventUserUpdated,
		Payload: map[string]string{"user_id": user.ID.String()},
	})

	return userToProfileResponse(user), nil
}

// DeleteAccountUseCase permanently removes a user.
type DeleteAccountUseCase struct {
	userRepo repository.UserRepository
	eventBus events.EventBus
}

// NewDeleteAccountUseCase constructs a DeleteAccountUseCase.
func NewDeleteAccountUseCase(userRepo repository.UserRepository, eventBus events.EventBus) *DeleteAccountUseCase {
	return &DeleteAccountUseCase{userRepo: userRepo, eventBus: eventBus}
}

// Execute deletes the user account.
func (uc *DeleteAccountUseCase) Execute(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return domainErr.ErrUserNotFound
	}

	if err := uc.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	_ = uc.eventBus.Publish(ctx, events.TopicUserDeleted, events.Event{
		Type:    event.EventUserDeleted,
		Payload: map[string]string{"user_id": userID},
	})
	return nil
}

// userToProfileResponse maps a domain User to the UserProfileResponse DTO.
func userToProfileResponse(u *model.User) *dto.UserProfileResponse {
	return &dto.UserProfileResponse{
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
