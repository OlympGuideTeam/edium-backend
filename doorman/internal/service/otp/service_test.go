package otpsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"doorman/internal/domain"
	otphandler "doorman/internal/handler/otp"
	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/pkg/apperr"
)

// --- mocks ---

type mockIdentityStore struct {
	identity *domain.Identity
	err      error
}

func (m *mockIdentityStore) Create(_ context.Context, _ string) (domain.Identity, error) {
	if m.err != nil {
		return domain.Identity{}, m.err
	}
	if m.identity != nil {
		return *m.identity, nil
	}
	return domain.Identity{}, nil
}
func (m *mockIdentityStore) GetByPhone(_ context.Context, _ string) (*domain.Identity, error) {
	return m.identity, m.err
}

type mockRegTokenStore struct {
	saveErr error
}

func (m *mockRegTokenStore) Save(_ context.Context, _, _ string, _ time.Duration) error {
	return m.saveErr
}

type mockOTPStore struct {
	exists        bool
	existsErr     error
	ttl           time.Duration
	ttlErr        error
	data          *OtpData
	getErr        error
	saveErr       error
	deleteErr     error
	incrAttempts  error
	sendCount     int64
	sendCountErr  error
}

func (m *mockOTPStore) Exists(_ context.Context, _ string) (bool, error) {
	return m.exists, m.existsErr
}
func (m *mockOTPStore) TTL(_ context.Context, _ string) (time.Duration, error) {
	return m.ttl, m.ttlErr
}
func (m *mockOTPStore) Get(_ context.Context, _ string) (*OtpData, error) {
	return m.data, m.getErr
}
func (m *mockOTPStore) Save(_ context.Context, _, _ string, _ time.Duration) error {
	return m.saveErr
}
func (m *mockOTPStore) Delete(_ context.Context, _ string) error     { return m.deleteErr }
func (m *mockOTPStore) IncrAttempts(_ context.Context, _ string) error { return m.incrAttempts }
func (m *mockOTPStore) IncrSendCount(_ context.Context, _ string) (int64, error) {
	return m.sendCount, m.sendCountErr
}

type mockTaskScheduler struct{ err error }

func (m *mockTaskScheduler) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.err
}

type mockJWTPublisher struct {
	access  string
	refresh string
	expires int64
	err     error
}

func (m *mockJWTPublisher) IssueTokens(_ context.Context, _ string) (string, string, int64, error) {
	return m.access, m.refresh, m.expires, m.err
}

const testPhone = "+71234567890"

func newSvc(id *mockIdentityStore, otp *mockOTPStore, reg *mockRegTokenStore, task *mockTaskScheduler, jwt *mockJWTPublisher, testPhones []string) *Service {
	return NewService(id, reg, otp, task, jwt, testPhones)
}

// --- SendOTP ---

func TestSendOTP_AlreadySent(t *testing.T) {
	otp := &mockOTPStore{exists: true, ttl: 2 * time.Minute}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	ttl, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if !errors.Is(err, apperr.ErrOTPAlreadySent) {
		t.Fatalf("expected ErrOTPAlreadySent, got %v", err)
	}
	if ttl != 2*time.Minute {
		t.Errorf("expected 2min TTL, got %v", ttl)
	}
}

func TestSendOTP_AlreadySent_TTLError(t *testing.T) {
	otp := &mockOTPStore{exists: true, ttlErr: errors.New("redis error")}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendOTP_SMS_IncrSendCountError(t *testing.T) {
	otp := &mockOTPStore{sendCountErr: errors.New("redis error")}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelSMS)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendOTP_SMS_DailyLimitExceeded(t *testing.T) {
	otp := &mockOTPStore{sendCount: maxDailySMSSends + 1}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelSMS)
	if !errors.Is(err, apperr.ErrOTPDailyLimit) {
		t.Fatalf("expected ErrOTPDailyLimit, got %v", err)
	}
}

func TestSendOTP_IdentityLookupError(t *testing.T) {
	id := &mockIdentityStore{err: errors.New("db error")}
	svc := newSvc(id, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendOTP_IdentityBlocked(t *testing.T) {
	id := &mockIdentityStore{identity: &domain.Identity{Status: domain.IdentityStatusBlocked}}
	svc := newSvc(id, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if !errors.Is(err, apperr.ErrPhoneUnavailable) {
		t.Fatalf("expected ErrPhoneUnavailable, got %v", err)
	}
}

func TestSendOTP_IdentityDeleted(t *testing.T) {
	id := &mockIdentityStore{identity: &domain.Identity{Status: domain.IdentityStatusDeleted}}
	svc := newSvc(id, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if !errors.Is(err, apperr.ErrPhoneUnavailable) {
		t.Fatalf("expected ErrPhoneUnavailable, got %v", err)
	}
}

func TestSendOTP_TestPhone_NoTaskScheduled(t *testing.T) {
	task := &mockTaskScheduler{}
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, task, &mockJWTPublisher{}, []string{testPhone})

	ttl, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ttl != otpTTL {
		t.Errorf("expected otpTTL, got %v", ttl)
	}
}

func TestSendOTP_OTPStoreSaveError(t *testing.T) {
	otp := &mockOTPStore{saveErr: errors.New("redis error")}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, []string{testPhone})

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendOTP_ScheduleError(t *testing.T) {
	task := &mockTaskScheduler{err: errors.New("db error")}
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, task, &mockJWTPublisher{}, nil)

	_, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendOTP_Success(t *testing.T) {
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	ttl, err := svc.SendOTP(context.Background(), testPhone, domain.ChannelTG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ttl != otpTTL {
		t.Errorf("expected %v, got %v", otpTTL, ttl)
	}
}

// --- VerifyOTP ---

func hashFor(svc *Service, otp uint64) string { return svc.hashOTP(otp) }

func TestVerifyOTP_GetError(t *testing.T) {
	otp := &mockOTPStore{getErr: errors.New("redis error")}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 123456)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyOTP_NotFound(t *testing.T) {
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 123456)
	if !errors.Is(err, apperr.ErrOTPNotFoundOrExpired) {
		t.Fatalf("expected ErrOTPNotFoundOrExpired, got %v", err)
	}
}

func TestVerifyOTP_AttemptsExceeded(t *testing.T) {
	otp := &mockOTPStore{data: &OtpData{Hash: "x", Attempts: maxOTPAttempts}}
	svc := newSvc(&mockIdentityStore{}, otp, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 123456)
	if !errors.Is(err, apperr.ErrOTPAttemptsExceeded) {
		t.Fatalf("expected ErrOTPAttemptsExceeded, got %v", err)
	}
}

func TestVerifyOTP_Invalid_IncrAttemptsError(t *testing.T) {
	otpStore := &mockOTPStore{
		data:         &OtpData{Hash: "wrong-hash", Attempts: 0},
		incrAttempts: errors.New("redis error"),
	}
	svc := newSvc(&mockIdentityStore{}, otpStore, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 123456)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyOTP_Invalid(t *testing.T) {
	otpStore := &mockOTPStore{data: &OtpData{Hash: "wrong-hash", Attempts: 0}}
	svc := newSvc(&mockIdentityStore{}, otpStore, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 123456)
	if !errors.Is(err, apperr.ErrOTPInvalid) {
		t.Fatalf("expected ErrOTPInvalid, got %v", err)
	}
}

func TestVerifyOTP_DeleteError(t *testing.T) {
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)
	hash := hashFor(svc, 111111)
	otpStore := &mockOTPStore{data: &OtpData{Hash: hash}, deleteErr: errors.New("redis error")}
	svc = newSvc(&mockIdentityStore{}, otpStore, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 111111)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyOTP_NewUser_ReturnsRegToken(t *testing.T) {
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)
	hash := hashFor(svc, 111111)
	otpStore := &mockOTPStore{data: &OtpData{Hash: hash}}
	svc = newSvc(&mockIdentityStore{identity: nil}, otpStore, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	result, err := svc.VerifyOTP(context.Background(), testPhone, 111111)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(*otphandler.RegistrationToken); !ok {
		t.Fatalf("expected RegistrationToken, got %T", result)
	}
}

func TestVerifyOTP_ExistingUser_ReturnsAuthTokens(t *testing.T) {
	identity := &domain.Identity{Status: domain.IdentityStatusActive}
	jwt := &mockJWTPublisher{access: "acc", refresh: "ref", expires: 900}
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, jwt, nil)
	hash := hashFor(svc, 111111)
	otpStore := &mockOTPStore{data: &OtpData{Hash: hash}}
	svc = newSvc(&mockIdentityStore{identity: identity}, otpStore, &mockRegTokenStore{}, &mockTaskScheduler{}, jwt, nil)

	result, err := svc.VerifyOTP(context.Background(), testPhone, 111111)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tokens, ok := result.(*tokenhandler.AuthTokens)
	if !ok {
		t.Fatalf("expected AuthTokens, got %T", result)
	}
	if tokens.AccessToken != "acc" {
		t.Errorf("unexpected access token: %s", tokens.AccessToken)
	}
}

func TestVerifyOTP_BlockedUser(t *testing.T) {
	identity := &domain.Identity{Status: domain.IdentityStatusBlocked}
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)
	hash := hashFor(svc, 111111)
	otpStore := &mockOTPStore{data: &OtpData{Hash: hash}}
	svc = newSvc(&mockIdentityStore{identity: identity}, otpStore, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 111111)
	if !errors.Is(err, apperr.ErrPhoneUnavailable) {
		t.Fatalf("expected ErrPhoneUnavailable, got %v", err)
	}
}

func TestVerifyOTP_RegTokenSaveError(t *testing.T) {
	svc := newSvc(&mockIdentityStore{}, &mockOTPStore{}, &mockRegTokenStore{}, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)
	hash := hashFor(svc, 111111)
	otpStore := &mockOTPStore{data: &OtpData{Hash: hash}}
	reg := &mockRegTokenStore{saveErr: errors.New("redis error")}
	svc = newSvc(&mockIdentityStore{identity: nil}, otpStore, reg, &mockTaskScheduler{}, &mockJWTPublisher{}, nil)

	_, err := svc.VerifyOTP(context.Background(), testPhone, 111111)
	if err == nil {
		t.Fatal("expected error")
	}
}
