package user

import (
	"context"
	"encoding/json"
	"fmt"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

type Service struct {
	users userStore
	tasks taskScheduler
	tx    txRunner
}

func NewService(users userStore, tasks taskScheduler, tx txRunner) *Service {
	return &Service{users: users, tasks: tasks, tx: tx}
}

func (s *Service) getActiveUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	if u == nil {
		return nil, apperr.ErrUserNotFound
	}
	if u.Status != domain.UserStatusActive {
		return nil, apperr.ErrUserBlockedOrDeleted
	}
	return u, nil
}

func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.getActiveUser(ctx, userID)
}

func (s *Service) UpdateMe(ctx context.Context, userID uuid.UUID, name, surname *string) error {
	u, err := s.getActiveUser(ctx, userID)
	if err != nil {
		return err
	}
	newName := u.Name
	newSurname := u.Surname
	if name != nil {
		newName = *name
	}
	if surname != nil {
		newSurname = *surname
	}
	if err := s.users.Update(ctx, userID, newName, newSurname); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

type userDeletedPayload struct {
	UserID string `json:"user_id"`
}

func (s *Service) GetUsersRoster(ctx context.Context, ids []uuid.UUID) ([]domain.User, error) {
	return s.users.GetManyByIDs(ctx, ids)
}

func (s *Service) GetMeStatistic(ctx context.Context, userID uuid.UUID) (*domain.UserStatistic, error) {
	if _, err := s.getActiveUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.users.GetStatistic(ctx, userID)
}

func (s *Service) DeleteMe(ctx context.Context, userID uuid.UUID) error {
	if _, err := s.getActiveUser(ctx, userID); err != nil {
		return err
	}
	payload, err := json.Marshal(userDeletedPayload{UserID: userID.String()})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.users.UpdateStatus(ctx, userID, domain.UserStatusDeleted); err != nil {
			return fmt.Errorf("UpdateStatus: %w", err)
		}
		if err := s.tasks.Schedule(ctx, domain.UserDeleted, payload); err != nil {
			return fmt.Errorf("schedule: %w", err)
		}
		return nil
	})
}
