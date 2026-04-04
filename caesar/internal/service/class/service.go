package class

import (
	"context"
	"fmt"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	classes classStore
}

func NewService(classes classStore) *Service {
	return &Service{classes: classes}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (s *Service) getClass(ctx context.Context, classID uuid.UUID) (*domain.ClassListItem, error) {
	c, err := s.classes.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("getByID: %w", err)
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) requireOwner(c *domain.ClassListItem, userID uuid.UUID) error {
	if c.OwnerID != userID {
		return ErrForbidden
	}
	return nil
}

// ─── Classes ──────────────────────────────────────────────────────────────────

func (s *Service) GetMyClasses(ctx context.Context, userID uuid.UUID, role domain.ClassMemberRole) ([]domain.ClassListItem, error) {
	classes, err := s.classes.ListByUserID(ctx, userID, role)
	if err != nil {
		return nil, fmt.Errorf("listByUserID: %w", err)
	}
	return classes, nil
}

func (s *Service) CreateClass(ctx context.Context, ownerID uuid.UUID, title string) (uuid.UUID, error) {
	if title == "" {
		return uuid.Nil, ErrEmptyTitle
	}
	id, err := s.classes.Create(ctx, ownerID, title)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create: %w", err)
	}
	return id, nil
}

func (s *Service) GetClass(ctx context.Context, classID, userID uuid.UUID) (*domain.ClassDetail, error) {
	c, err := s.getClass(ctx, classID)
	if err != nil {
		return nil, err
	}

	members, err := s.classes.GetMembersForDetail(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("getMembersForDetail: %w", err)
	}

	detail := &domain.ClassDetail{
		Class:     c.Class,
		OwnerName: c.OwnerName,
		IsOwner:   c.OwnerID == userID,
	}
	for _, m := range members {
		switch m.Role {
		case domain.ClassMemberRoleTeacher:
			detail.Teachers = append(detail.Teachers, m)
		case domain.ClassMemberRoleStudent:
			detail.Students = append(detail.Students, m)
		}
	}
	return detail, nil
}

func (s *Service) UpdateClass(ctx context.Context, classID, userID uuid.UUID, title string) error {
	if title == "" {
		return ErrEmptyTitle
	}
	c, err := s.getClass(ctx, classID)
	if err != nil {
		return err
	}
	if err := s.requireOwner(c, userID); err != nil {
		return err
	}
	if err := s.classes.Update(ctx, classID, title); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (s *Service) DeleteClass(ctx context.Context, classID, userID uuid.UUID) error {
	c, err := s.getClass(ctx, classID)
	if err != nil {
		return err
	}
	if err := s.requireOwner(c, userID); err != nil {
		return err
	}
	if err := s.classes.Delete(ctx, classID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// ─── Members ──────────────────────────────────────────────────────────────────

func (s *Service) RemoveMember(ctx context.Context, classID, userID, targetUserID uuid.UUID) error {
	c, err := s.getClass(ctx, classID)
	if err != nil {
		return err
	}
	if err := s.requireOwner(c, userID); err != nil {
		return err
	}
	if targetUserID == c.OwnerID {
		return ErrRemoveOwner
	}
	if err := s.classes.RemoveMember(ctx, classID, targetUserID); err != nil {
		return err
	}
	return nil
}
