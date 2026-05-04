package course

import (
	"context"
	"fmt"
	"strings"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

type Service struct {
	courses courseStore
	classes classAccessor
	sheets  sheetAccessor
	tasks   taskScheduler
	tx      txRunner
}

func NewService(courses courseStore, classes classAccessor, tasks taskScheduler, tx txRunner) *Service {
	return &Service{courses: courses, classes: classes, tasks: tasks, tx: tx}
}

func (s *Service) CreateCourse(ctx context.Context, classID, userID uuid.UUID, title string) (uuid.UUID, error) {
	if strings.TrimSpace(title) == "" {
		return uuid.Nil, apperr.ErrCourseEmptyTitle
	}

	role, ok, err := s.classes.GetMemberRole(ctx, classID, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return uuid.Nil, apperr.ErrCourseNotMember
	}
	if role == domain.ClassMemberRoleStudent {
		return uuid.Nil, apperr.ErrCourseForbidden
	}

	id, err := s.courses.Create(ctx, classID, userID, title)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create: %w", err)
	}
	return id, nil
}

func (s *Service) GetCourse(ctx context.Context, courseID, userID uuid.UUID) (*domain.CourseDetail, error) {
	detail, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("getDetail: %w", err)
	}
	if detail == nil {
		return nil, apperr.ErrCourseNotFound
	}

	role, ok, err := s.classes.GetMemberRole(ctx, detail.ClassID, userID)
	if err != nil {
		return nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return nil, apperr.ErrCourseNotMember
	}

	modules, err := s.courses.ListModules(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("listModules: %w", err)
	}

	drafts, err := s.courses.ListDraftsByCourseID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("listDrafts: %w", err)
	}

	detail.Drafts = drafts
	detail.Modules = modules
	detail.IsTeacher = role != domain.ClassMemberRoleStudent

	return detail, nil
}

func (s *Service) GetClassCourses(ctx context.Context, classID, userID uuid.UUID) ([]domain.CourseListItem, error) {
	cl, err := s.classes.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("getClass: %w", err)
	}
	if cl == nil {
		return nil, apperr.ErrCourseNotFound
	}

	role, ok, err := s.classes.GetMemberRole(ctx, classID, userID)
	if err != nil {
		return nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return nil, apperr.ErrCourseNotMember
	}

	isTeacher := role != domain.ClassMemberRoleStudent

	items, err := s.courses.ListByClassID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("listByClassID: %w", err)
	}
	for i := range items {
		items[i].IsTeacher = isTeacher
	}
	return items, nil
}

func (s *Service) UpdateCourse(ctx context.Context, courseID, userID uuid.UUID, title string) error {
	if strings.TrimSpace(title) == "" {
		return apperr.ErrCourseEmptyTitle
	}

	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("getByID: %w", err)
	}
	if course == nil {
		return apperr.ErrCourseNotFound
	}

	role, ok, err := s.classes.GetMemberRole(ctx, course.ClassID, userID)
	if err != nil {
		return fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok || role == domain.ClassMemberRoleStudent {
		return apperr.ErrCourseForbidden
	}

	if err := s.courses.Update(ctx, courseID, title); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (s *Service) GetCourseSheet(ctx context.Context, courseID, userID uuid.UUID) (*domain.CourseSheet, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("getByID: %w", err)
	}
	if course == nil {
		return nil, apperr.ErrCourseNotFound
	}

	_, ok, err := s.classes.GetMemberRole(ctx, course.ClassID, userID)
	if err != nil {
		return nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return nil, apperr.ErrCourseNotMember
	}

	items, err := s.courses.GetSheetItems(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("getSheetItems: %w", err)
	}

	scores, err := s.courses.GetSheetScores(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("getSheetScores: %w", err)
	}

	members, err := s.classes.GetMembersForDetail(ctx, course.ClassID)
	if err != nil {
		return nil, fmt.Errorf("getMembersForDetail: %w", err)
	}

	var students []domain.ClassMember
	for _, m := range members {
		if m.Role == domain.ClassMemberRoleStudent {
			students = append(students, m)
		}
	}

	itemIDToIndex := make(map[uuid.UUID]int)
	for i, item := range items {
		itemIDToIndex[item.ID] = i
	}

	userIDToScores := make(map[uuid.UUID][]domain.SheetScore)
	for _, s := range scores {
		userIDToScores[s.UserID] = append(userIDToScores[s.UserID], domain.SheetScore{
			ItemID: s.ItemID,
			Score:  s.Score,
		})
	}

	sheetRows := make([]domain.SheetRow, 0, len(students))
	for _, student := range students {
		rowScores := make([]domain.SheetScore, len(items))
		for idx := range items {
			rowScores[idx].ItemID = items[idx].ID
		}

		if userScores, exists := userIDToScores[student.UserID]; exists {
			for _, sc := range userScores {
				if idx, ok := itemIDToIndex[sc.ItemID]; ok {
					rowScores[idx] = sc
				}
			}
		}

		sheetRows = append(sheetRows, domain.SheetRow{
			StudentID:   student.UserID,
			StudentName: student.Name + " " + student.Surname,
			Scores:      rowScores,
		})
	}

	return &domain.CourseSheet{
		Items:    items,
		Students: sheetRows,
	}, nil
}

func (s *Service) DeleteCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	c, err := s.getCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}

	if err := s.courses.Delete(ctx, courseID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (s *Service) getCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	c, err := s.courses.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getByID: %w", err)
	}
	if c == nil {
		return nil, apperr.ErrCourseNotFound
	}
	return c, nil
}

// requireCanModify проверяет, что пользователь — создатель курса или владелец класса.
func (s *Service) requireCanModify(ctx context.Context, c *domain.Course, userID uuid.UUID) error {
	if c.OwnerID == userID {
		return nil
	}
	cl, err := s.classes.GetByID(ctx, c.ClassID)
	if err != nil {
		return fmt.Errorf("getClass: %w", err)
	}
	if cl == nil || cl.OwnerID != userID {
		return apperr.ErrCourseForbidden
	}
	return nil
}
