package jwtsvc

import (
	"context"
	"crypto/rsa"
	"errors"
	"math/big"
	"testing"
	"time"

	"doorman/internal/domain"
	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/pkg/apperr"
	"doorman/internal/transport/dto"
)

type mockKeyStore struct {
	keys          map[string]*rsa.PublicKey
	authTokens    *AuthTokensData
	genErr        error
	parsedClaims  *RefreshClaims
	parseErr      error
}

func (m *mockKeyStore) GetPublicKeys() map[string]*rsa.PublicKey { return m.keys }
func (m *mockKeyStore) GenerateAuthTokens(_ string, _, _ time.Duration) (*AuthTokensData, error) {
	return m.authTokens, m.genErr
}
func (m *mockKeyStore) ParseRefreshToken(_ string) (*RefreshClaims, error) {
	return m.parsedClaims, m.parseErr
}

type mockRefreshStore struct {
	savedUserID string
	saveErr     error
	getResult   string
	getErr      error
}

func (m *mockRefreshStore) SaveToken(_ context.Context, _, userID string, _ time.Duration) error {
	m.savedUserID = userID
	return m.saveErr
}
func (m *mockRefreshStore) GetAndDelToken(_ context.Context, _ string) (string, error) {
	return m.getResult, m.getErr
}

type mockTaskRepo struct{ err error }

func (m *mockTaskRepo) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.err
}

func TestGetPublicKeys_Empty(t *testing.T) {
	svc := NewService(&mockKeyStore{keys: map[string]*rsa.PublicKey{}}, &mockRefreshStore{}, &mockTaskRepo{})
	resp := svc.GetPublicKeys()
	if len(resp.Keys) != 0 {
		t.Errorf("expected empty keys, got %d", len(resp.Keys))
	}
}

func TestGetPublicKeys_ReturnsJWK(t *testing.T) {
	keys := map[string]*rsa.PublicKey{
		"key1": {N: big.NewInt(65537), E: 65537},
	}
	svc := NewService(&mockKeyStore{keys: keys}, &mockRefreshStore{}, &mockTaskRepo{})
	resp := svc.GetPublicKeys()
	if len(resp.Keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(resp.Keys))
	}
	if resp.Keys[0].KID != "key1" {
		t.Errorf("unexpected kid: %s", resp.Keys[0].KID)
	}
	_ = dto.JWKSResponse{}
}

func TestIssueTokens_GenerateError(t *testing.T) {
	ks := &mockKeyStore{genErr: errors.New("key error")}
	svc := NewService(ks, &mockRefreshStore{}, &mockTaskRepo{})

	_, _, _, err := svc.IssueTokens(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIssueTokens_SaveTokenError(t *testing.T) {
	ks := &mockKeyStore{authTokens: &AuthTokensData{AccessToken: "acc", RefreshToken: "ref", RefreshJti: "jti", ExpiresIn: 900}}
	rs := &mockRefreshStore{saveErr: errors.New("redis error")}
	svc := NewService(ks, rs, &mockTaskRepo{})

	_, _, _, err := svc.IssueTokens(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIssueTokens_Success(t *testing.T) {
	ks := &mockKeyStore{authTokens: &AuthTokensData{AccessToken: "acc", RefreshToken: "ref", RefreshJti: "jti", ExpiresIn: 900}}
	svc := NewService(ks, &mockRefreshStore{}, &mockTaskRepo{})

	acc, ref, exp, err := svc.IssueTokens(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acc != "acc" || ref != "ref" || exp != 900 {
		t.Errorf("unexpected tokens: %s %s %d", acc, ref, exp)
	}
}

func TestLogout_ParseError(t *testing.T) {
	ks := &mockKeyStore{parseErr: errors.New("invalid token")}
	svc := NewService(ks, &mockRefreshStore{}, &mockTaskRepo{})

	if err := svc.Logout(context.Background(), "bad-token"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLogout_GetAndDelError(t *testing.T) {
	ks := &mockKeyStore{parsedClaims: &RefreshClaims{}}
	rs := &mockRefreshStore{getErr: errors.New("redis error")}
	svc := NewService(ks, rs, &mockTaskRepo{})

	if err := svc.Logout(context.Background(), "token"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLogout_SubjectMismatch(t *testing.T) {
	ks := &mockKeyStore{parsedClaims: &RefreshClaims{}}
	ks.parsedClaims.Subject = "user-A"
	rs := &mockRefreshStore{getResult: "user-B"}
	svc := NewService(ks, rs, &mockTaskRepo{})

	err := svc.Logout(context.Background(), "token")
	if !errors.Is(err, apperr.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
	}
}

func TestLogout_Success_ScheduleErrorIsLogged(t *testing.T) {
	ks := &mockKeyStore{parsedClaims: &RefreshClaims{}}
	ks.parsedClaims.Subject = "user-1"
	rs := &mockRefreshStore{getResult: "user-1"}
	task := &mockTaskRepo{err: errors.New("db error")}
	svc := NewService(ks, rs, task)

	if err := svc.Logout(context.Background(), "token"); err != nil {
		t.Fatalf("schedule error should be logged, not returned: %v", err)
	}
}

func TestLogout_Success(t *testing.T) {
	ks := &mockKeyStore{parsedClaims: &RefreshClaims{}}
	ks.parsedClaims.Subject = "user-1"
	rs := &mockRefreshStore{getResult: "user-1"}
	svc := NewService(ks, rs, &mockTaskRepo{})

	if err := svc.Logout(context.Background(), "token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefresh_ParseError(t *testing.T) {
	ks := &mockKeyStore{parseErr: errors.New("invalid")}
	svc := NewService(ks, &mockRefreshStore{}, &mockTaskRepo{})

	_, err := svc.Refresh(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRefresh_SubjectMismatch(t *testing.T) {
	ks := &mockKeyStore{parsedClaims: &RefreshClaims{}}
	ks.parsedClaims.Subject = "user-A"
	rs := &mockRefreshStore{getResult: "user-B"}
	svc := NewService(ks, rs, &mockTaskRepo{})

	_, err := svc.Refresh(context.Background(), "token")
	if !errors.Is(err, apperr.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
	}
}

func TestRefresh_IssueError(t *testing.T) {
	ks := &mockKeyStore{
		parsedClaims: &RefreshClaims{},
		genErr:       errors.New("key error"),
	}
	ks.parsedClaims.Subject = "user-1"
	rs := &mockRefreshStore{getResult: "user-1"}
	svc := NewService(ks, rs, &mockTaskRepo{})

	_, err := svc.Refresh(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRefresh_Success(t *testing.T) {
	ks := &mockKeyStore{
		parsedClaims: &RefreshClaims{},
		authTokens:   &AuthTokensData{AccessToken: "new-acc", RefreshToken: "new-ref", ExpiresIn: 900},
	}
	ks.parsedClaims.Subject = "user-1"
	rs := &mockRefreshStore{getResult: "user-1"}
	svc := NewService(ks, rs, &mockTaskRepo{})

	tokens, err := svc.Refresh(context.Background(), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := interface{}(tokens).(*tokenhandler.AuthTokens); !ok {
		t.Fatalf("expected *AuthTokens, got %T", tokens)
	}
	if tokens.AccessToken != "new-acc" {
		t.Errorf("unexpected access token: %s", tokens.AccessToken)
	}
}
