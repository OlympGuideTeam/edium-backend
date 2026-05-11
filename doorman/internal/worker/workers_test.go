package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"doorman/internal/domain"
	"doorman/internal/pkg/apperr"
)

// --- shared mocks ---

type mockTaskRepo struct {
	markDoneErr   error
	markFailedErr error
	scheduleErr   error
}

func (m *mockTaskRepo) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.scheduleErr
}
func (m *mockTaskRepo) ClaimPending(_ context.Context, _ domain.TaskType, _ int) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) MarkDone(_ context.Context, _ uuid.UUID) error { return m.markDoneErr }
func (m *mockTaskRepo) MarkFailed(_ context.Context, _ uuid.UUID, _ string, _ time.Duration) error {
	return m.markFailedErr
}

type mockOTPSender struct {
	ttl time.Duration
	err error
}

func (m *mockOTPSender) SendOTP(_ context.Context, _ string, _ domain.Channel) (time.Duration, error) {
	return m.ttl, m.err
}

type mockIdentityUpdater struct{ err error }

func (m *mockIdentityUpdater) UpdateStatus(_ context.Context, _ string, _ domain.IdentityStatus) error {
	return m.err
}

type mockTokensCleaner struct{ err error }

func (m *mockTokensCleaner) DeleteUserTokens(_ context.Context, _ string) error { return m.err }

type mockScheduler struct {
	calls []domain.TaskType
	err   error
}

func (m *mockScheduler) Schedule(_ context.Context, tt domain.TaskType, _ []byte) error {
	m.calls = append(m.calls, tt)
	return m.err
}

func makeTask(payload any) domain.Task {
	b, _ := json.Marshal(payload)
	return domain.Task{ID: uuid.New(), Payload: b}
}

// --- OTPRequestConsumer.handle ---

func TestOTPRequestConsumer_Success(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &OTPRequestConsumer{tasks: scheduler}

	msg := otpRequestMsg{Phone: "+71234567890", Channel: domain.ChannelTG}
	if err := c.handle(context.Background(), mustMarshal(msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 1 || scheduler.calls[0] != domain.OTPRequest {
		t.Errorf("expected OTPRequest scheduled, got %v", scheduler.calls)
	}
}

func TestOTPRequestConsumer_BadJSON(t *testing.T) {
	c := &OTPRequestConsumer{tasks: &mockScheduler{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- OTPRequestProcessor.processTask ---

func TestOTPRequestProcessor_Success(t *testing.T) {
	tasks := &mockTaskRepo{}
	svc := &mockOTPSender{ttl: 3 * time.Minute}
	w := NewOTPRequestProcessor(tasks, svc)

	t1 := makeTask(map[string]any{"phone": "+71234567890", "channel": "tg"})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPRequestProcessor_OTPAlreadySent_SchedulesError(t *testing.T) {
	tasks := &mockTaskRepo{}
	svc := &mockOTPSender{err: apperr.ErrOTPAlreadySent}
	w := NewOTPRequestProcessor(tasks, svc)

	t1 := makeTask(map[string]any{"phone": "+71234567890", "channel": "tg"})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPRequestProcessor_PhoneUnavailable_SchedulesError(t *testing.T) {
	tasks := &mockTaskRepo{}
	svc := &mockOTPSender{err: apperr.ErrPhoneUnavailable}
	w := NewOTPRequestProcessor(tasks, svc)

	t1 := makeTask(map[string]any{"phone": "+71234567890", "channel": "tg"})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPRequestProcessor_DailyLimit_SchedulesError(t *testing.T) {
	tasks := &mockTaskRepo{}
	svc := &mockOTPSender{err: apperr.ErrOTPDailyLimit}
	w := NewOTPRequestProcessor(tasks, svc)

	t1 := makeTask(map[string]any{"phone": "+71234567890", "channel": "sms"})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPRequestProcessor_OtherError_Returns(t *testing.T) {
	tasks := &mockTaskRepo{}
	svc := &mockOTPSender{err: errors.New("db error")}
	w := NewOTPRequestProcessor(tasks, svc)

	t1 := makeTask(map[string]any{"phone": "+71234567890", "channel": "tg"})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestOTPRequestProcessor_BadPayload(t *testing.T) {
	w := NewOTPRequestProcessor(&mockTaskRepo{}, &mockOTPSender{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- UserDeletedConsumer.handle ---

func TestUserDeletedConsumer_Success(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &UserDeletedConsumer{tasks: scheduler}

	msg := userDeletedMsg{UserID: uuid.New().String()}
	if err := c.handle(context.Background(), mustMarshal(msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 1 || scheduler.calls[0] != domain.UserDeleted {
		t.Errorf("expected UserDeleted scheduled, got %v", scheduler.calls)
	}
}

func TestUserDeletedConsumer_BadJSON(t *testing.T) {
	c := &UserDeletedConsumer{tasks: &mockScheduler{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- UserDeletedProcessor.processTask ---

func TestUserDeletedProcessor_Success(t *testing.T) {
	w := NewUserDeletedProcessor(&mockTaskRepo{}, &mockIdentityUpdater{}, &mockTokensCleaner{})
	t1 := makeTask(map[string]string{"user_id": uuid.New().String()})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserDeletedProcessor_UpdateStatusError(t *testing.T) {
	id := &mockIdentityUpdater{err: errors.New("db error")}
	w := NewUserDeletedProcessor(&mockTaskRepo{}, id, &mockTokensCleaner{})
	t1 := makeTask(map[string]string{"user_id": uuid.New().String()})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestUserDeletedProcessor_DeleteTokensError(t *testing.T) {
	tokens := &mockTokensCleaner{err: errors.New("redis error")}
	w := NewUserDeletedProcessor(&mockTaskRepo{}, &mockIdentityUpdater{}, tokens)
	t1 := makeTask(map[string]string{"user_id": uuid.New().String()})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestUserDeletedProcessor_BadPayload(t *testing.T) {
	w := NewUserDeletedProcessor(&mockTaskRepo{}, &mockIdentityUpdater{}, &mockTokensCleaner{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
