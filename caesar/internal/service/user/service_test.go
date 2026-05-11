package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"
)

type mockUserStore struct {
	user         *domain.User
	getErr       error
	updateErr    error
	statusErr    error
	statistic    *domain.UserStatistic
	statisticErr error
	manyUsers    []domain.User
	manyErr      error
}

func (m *mockUserStore) GetByID(_ context.Context, _ uuid.UUID) (*domain.User, error) {
	return m.user, m.getErr
}
func (m *mockUserStore) GetManyByIDs(_ context.Context, _ []uuid.UUID) ([]domain.User, error) {
	return m.manyUsers, m.manyErr
}
func (m *mockUserStore) Update(_ context.Context, _ uuid.UUID, _, _ string) error {
	return m.updateErr
}
func (m *mockUserStore) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.UserStatus) error {
	return m.statusErr
}
func (m *mockUserStore) GetStatistic(_ context.Context, _ uuid.UUID) (*domain.UserStatistic, error) {
	return m.statistic, m.statisticErr
}

type mockTaskScheduler struct{ err error }

func (m *mockTaskScheduler) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.err
}

type mockTx struct{ err error }

func (m *mockTx) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(ctx)
}

func activeUser() *domain.User {
	return &domain.User{ID: uuid.New(), Name: "Ivan", Surname: "Petrov", Status: domain.UserStatusActive}
}

func newSvc(us *mockUserStore, task *mockTaskScheduler, tx *mockTx) *Service {
	return NewService(us, task, tx)
}

// --- GetMe ---

func TestGetMe_NotFound(t *testing.T) {
	svc := newSvc(&mockUserStore{}, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetMe(context.Background(), uuid.New())
	if !errors.Is(err, apperr.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetMe_Blocked(t *testing.T) {
	us := &mockUserStore{user: &domain.User{Status: domain.UserStatusBlocked}}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetMe(context.Background(), uuid.New())
	if !errors.Is(err, apperr.ErrUserBlockedOrDeleted) {
		t.Fatalf("expected ErrUserBlockedOrDeleted, got %v", err)
	}
}

func TestGetMe_Success(t *testing.T) {
	us := &mockUserStore{user: activeUser()}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	u, err := svc.GetMe(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "Ivan" {
		t.Errorf("unexpected name: %s", u.Name)
	}
}

// --- UpdateMe ---

func TestUpdateMe_NotFound(t *testing.T) {
	svc := newSvc(&mockUserStore{}, &mockTaskScheduler{}, &mockTx{})
	name := "Anna"
	if err := svc.UpdateMe(context.Background(), uuid.New(), &name, nil); !errors.Is(err, apperr.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateMe_UpdateError(t *testing.T) {
	us := &mockUserStore{user: activeUser(), updateErr: errors.New("db error")}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	name := "Anna"
	if err := svc.UpdateMe(context.Background(), uuid.New(), &name, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateMe_NameOnly(t *testing.T) {
	us := &mockUserStore{user: activeUser()}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	name := "Anna"
	if err := svc.UpdateMe(context.Background(), uuid.New(), &name, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateMe_SurnameOnly(t *testing.T) {
	us := &mockUserStore{user: activeUser()}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	surname := "Ivanova"
	if err := svc.UpdateMe(context.Background(), uuid.New(), nil, &surname); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetUsersRoster ---

func TestGetUsersRoster_Success(t *testing.T) {
	users := []domain.User{{ID: uuid.New(), Name: "Ivan"}}
	us := &mockUserStore{manyUsers: users}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	result, err := svc.GetUsersRoster(context.Background(), []uuid.UUID{uuid.New()})
	if err != nil || len(result) != 1 {
		t.Fatalf("unexpected: err=%v len=%d", err, len(result))
	}
}

// --- GetMeStatistic ---

func TestGetMeStatistic_UserNotFound(t *testing.T) {
	svc := newSvc(&mockUserStore{}, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetMeStatistic(context.Background(), uuid.New())
	if !errors.Is(err, apperr.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetMeStatistic_Success(t *testing.T) {
	stat := &domain.UserStatistic{ClassTeacherCount: 2, StudentCount: 10}
	us := &mockUserStore{user: activeUser(), statistic: stat}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	result, err := svc.GetMeStatistic(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClassTeacherCount != 2 {
		t.Errorf("unexpected statistic: %+v", result)
	}
}

// --- DeleteMe ---

func TestDeleteMe_NotFound(t *testing.T) {
	svc := newSvc(&mockUserStore{}, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteMe(context.Background(), uuid.New()); !errors.Is(err, apperr.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestDeleteMe_UpdateStatusError(t *testing.T) {
	us := &mockUserStore{user: activeUser(), statusErr: errors.New("db error")}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteMe(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteMe_ScheduleError(t *testing.T) {
	us := &mockUserStore{user: activeUser()}
	task := &mockTaskScheduler{err: errors.New("db error")}
	svc := newSvc(us, task, &mockTx{})
	if err := svc.DeleteMe(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteMe_Success(t *testing.T) {
	us := &mockUserStore{user: activeUser()}
	svc := newSvc(us, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteMe(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
