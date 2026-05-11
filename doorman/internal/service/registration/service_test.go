package regsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"doorman/internal/domain"
	"doorman/internal/pkg/apperr"
)

type mockIdentityStore struct {
	identity domain.Identity
	err      error
}

func (m *mockIdentityStore) Create(_ context.Context, _ string) (domain.Identity, error) {
	return m.identity, m.err
}

type mockRegTokenStore struct {
	stored    string
	getErr    error
	deleteErr error
}

func (m *mockRegTokenStore) Get(_ context.Context, _ string) (string, error) {
	return m.stored, m.getErr
}
func (m *mockRegTokenStore) Delete(_ context.Context, _ string) error { return m.deleteErr }

type mockTaskScheduler struct{ err error }

func (m *mockTaskScheduler) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.err
}

type mockTxManager struct{ err error }

func (m *mockTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(ctx)
}

type mockJWTIssuer struct {
	access  string
	refresh string
	expires int64
	err     error
}

func (m *mockJWTIssuer) IssueTokens(_ context.Context, _ string) (string, string, int64, error) {
	return m.access, m.refresh, m.expires, m.err
}

func newSvc(id *mockIdentityStore, reg *mockRegTokenStore, task *mockTaskScheduler, jwt *mockJWTIssuer, tx *mockTxManager) *Service {
	return NewService(id, reg, task, jwt, tx)
}

func TestRegister_TokenGetError(t *testing.T) {
	reg := &mockRegTokenStore{getErr: errors.New("redis error")}
	svc := newSvc(&mockIdentityStore{}, reg, &mockTaskScheduler{}, &mockJWTIssuer{}, &mockTxManager{})

	_, err := svc.Register(context.Background(), "+71234567890", "Ivan", "Petrov", "tok")
	if !errors.Is(err, apperr.ErrRegTokenInvalid) {
		t.Fatalf("expected ErrRegTokenInvalid, got %v", err)
	}
}

func TestRegister_TokenMismatch(t *testing.T) {
	reg := &mockRegTokenStore{stored: "correct-token"}
	svc := newSvc(&mockIdentityStore{}, reg, &mockTaskScheduler{}, &mockJWTIssuer{}, &mockTxManager{})

	_, err := svc.Register(context.Background(), "+71234567890", "Ivan", "Petrov", "wrong-token")
	if !errors.Is(err, apperr.ErrRegTokenInvalid) {
		t.Fatalf("expected ErrRegTokenInvalid, got %v", err)
	}
}

func TestRegister_CreateIdentityError(t *testing.T) {
	reg := &mockRegTokenStore{stored: "tok"}
	id := &mockIdentityStore{err: errors.New("db error")}
	svc := newSvc(id, reg, &mockTaskScheduler{}, &mockJWTIssuer{}, &mockTxManager{})

	_, err := svc.Register(context.Background(), "+71234567890", "Ivan", "Petrov", "tok")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_ScheduleError(t *testing.T) {
	reg := &mockRegTokenStore{stored: "tok"}
	id := &mockIdentityStore{identity: domain.Identity{ID: uuid.New()}}
	task := &mockTaskScheduler{err: errors.New("db error")}
	svc := newSvc(id, reg, task, &mockJWTIssuer{}, &mockTxManager{})

	_, err := svc.Register(context.Background(), "+71234567890", "Ivan", "Petrov", "tok")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_JWTIssueError(t *testing.T) {
	reg := &mockRegTokenStore{stored: "tok"}
	id := &mockIdentityStore{identity: domain.Identity{ID: uuid.New()}}
	jwt := &mockJWTIssuer{err: errors.New("key error")}
	svc := newSvc(id, reg, &mockTaskScheduler{}, jwt, &mockTxManager{})

	_, err := svc.Register(context.Background(), "+71234567890", "Ivan", "Petrov", "tok")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_Success(t *testing.T) {
	reg := &mockRegTokenStore{stored: "tok"}
	id := &mockIdentityStore{identity: domain.Identity{ID: uuid.New()}}
	jwt := &mockJWTIssuer{access: "acc", refresh: "ref", expires: 900}
	svc := newSvc(id, reg, &mockTaskScheduler{}, jwt, &mockTxManager{})

	tokens, err := svc.Register(context.Background(), "+71234567890", "Ivan", "Petrov", "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.AccessToken != "acc" || tokens.RefreshToken != "ref" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}
}
