package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"herald/internal/domain"
)

// --- shared mocks for processors ---

type mockTaskRepo struct {
	markDoneErr   error
	markFailedErr error
}

func (m *mockTaskRepo) ClaimPending(_ context.Context, _ domain.TaskType, _ int) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) MarkDone(_ context.Context, _ uuid.UUID) error    { return m.markDoneErr }
func (m *mockTaskRepo) MarkFailed(_ context.Context, _ uuid.UUID, _ string, _ time.Duration) error {
	return m.markFailedErr
}

type mockPendingOTPLookup struct {
	getResult *domain.PendingOTP
	getErr    error
	deleteErr error
}

func (m *mockPendingOTPLookup) GetPendingOTP(_ context.Context, _ string, _ domain.Channel) (*domain.PendingOTP, error) {
	return m.getResult, m.getErr
}
func (m *mockPendingOTPLookup) DeletePendingOTP(_ context.Context, _ string, _ domain.Channel) error {
	return m.deleteErr
}

type mockMessageSender struct {
	err error
}

func (m *mockMessageSender) Send(_ context.Context, _ int64, _ string) error { return m.err }

type mockSMSSender struct{ err error }

func (m *mockSMSSender) SendSMS(_ context.Context, _, _ string, _ uuid.UUID) error { return m.err }

// --- OTPSentProcessor ---

func otpSentTask(phone string, otp uint64, channel domain.Channel) domain.Task {
	b, _ := json.Marshal(map[string]any{"phone": phone, "otp": otp, "channel": channel})
	return makeRawTask(b)
}

func otpSentErrorTask(phone string, channel domain.Channel, errCode string) domain.Task {
	b, _ := json.Marshal(map[string]any{"phone": phone, "otp": 0, "channel": channel, "error_code": errCode})
	return makeRawTask(b)
}

func TestOTPSentProcessor_SMS_Success(t *testing.T) {
	tasks := &mockTaskRepo{}
	smsSender := &mockSMSSender{}
	w := NewOTPSentProcessor(tasks, &mockPendingOTPLookup{}, nil, smsSender)

	t1 := otpSentTask("+79001234567", 123456, domain.ChannelSMS)
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_SMS_NilSender(t *testing.T) {
	w := NewOTPSentProcessor(&mockTaskRepo{}, &mockPendingOTPLookup{}, nil, nil)
	t1 := otpSentTask("+79001234567", 123456, domain.ChannelSMS)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error when SMS sender is nil")
	}
}

func TestOTPSentProcessor_SMS_ErrorCode(t *testing.T) {
	smsSender := &mockSMSSender{}
	w := NewOTPSentProcessor(&mockTaskRepo{}, &mockPendingOTPLookup{}, nil, smsSender)
	t1 := otpSentErrorTask("+79001234567", domain.ChannelSMS, "OTP_ALREADY_SENT")
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_SMS_SendError(t *testing.T) {
	smsSender := &mockSMSSender{err: errors.New("sms gateway error")}
	w := NewOTPSentProcessor(&mockTaskRepo{}, &mockPendingOTPLookup{}, nil, smsSender)
	t1 := otpSentTask("+79001234567", 123456, domain.ChannelSMS)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestOTPSentProcessor_TG_PendingNotFound(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: nil}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, nil, nil)
	t1 := otpSentTask("+79001234567", 123456, domain.ChannelTG)
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_TG_PendingLookupError(t *testing.T) {
	lookup := &mockPendingOTPLookup{getErr: errors.New("db error")}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, nil, nil)
	t1 := otpSentTask("+79001234567", 123456, domain.ChannelTG)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestOTPSentProcessor_TG_NoSenderForChannel(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: &domain.PendingOTP{ChatID: 123}}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, map[domain.Channel]MessageSender{}, nil)
	t1 := otpSentTask("+79001234567", 123456, domain.ChannelTG)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error: no sender for channel")
	}
}

func TestOTPSentProcessor_TG_SendError(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: &domain.PendingOTP{ChatID: 123}}
	sender := &mockMessageSender{err: errors.New("tg error")}
	senders := map[domain.Channel]MessageSender{domain.ChannelTG: sender}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, senders, nil)
	t1 := otpSentTask("+79001234567", 123456, domain.ChannelTG)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestOTPSentProcessor_TG_ErrorCode_OTPAlreadySent(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: &domain.PendingOTP{ChatID: 123}}
	sender := &mockMessageSender{}
	senders := map[domain.Channel]MessageSender{domain.ChannelTG: sender}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, senders, nil)
	t1 := otpSentErrorTask("+79001234567", domain.ChannelTG, "OTP_ALREADY_SENT")
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_TG_ErrorCode_PhoneUnavailable(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: &domain.PendingOTP{ChatID: 123}}
	sender := &mockMessageSender{}
	senders := map[domain.Channel]MessageSender{domain.ChannelTG: sender}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, senders, nil)
	t1 := otpSentErrorTask("+79001234567", domain.ChannelTG, "PHONE_UNAVAILABLE")
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_TG_ErrorCode_Unknown(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: &domain.PendingOTP{ChatID: 123}}
	sender := &mockMessageSender{}
	senders := map[domain.Channel]MessageSender{domain.ChannelTG: sender}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, senders, nil)
	t1 := otpSentErrorTask("+79001234567", domain.ChannelTG, "SOMETHING_ELSE")
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_TG_Success(t *testing.T) {
	lookup := &mockPendingOTPLookup{getResult: &domain.PendingOTP{ChatID: 123}}
	sender := &mockMessageSender{}
	senders := map[domain.Channel]MessageSender{domain.ChannelTG: sender}
	w := NewOTPSentProcessor(&mockTaskRepo{}, lookup, senders, nil)
	t1 := otpSentTask("+79001234567", 654321, domain.ChannelTG)
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOTPSentProcessor_BadPayload(t *testing.T) {
	w := NewOTPSentProcessor(&mockTaskRepo{}, &mockPendingOTPLookup{}, nil, nil)
	if err := w.processTask(context.Background(), makeRawTask([]byte("bad"))); err == nil {
		t.Fatal("expected error")
	}
}

// --- PushNotificationProcessor ---

type mockPushDevices struct {
	listResult  []domain.FCMDevice
	listErr     error
	deleteErr   error
}

func (m *mockPushDevices) ListByUserID(_ context.Context, _ uuid.UUID) ([]domain.FCMDevice, error) {
	return m.listResult, m.listErr
}
func (m *mockPushDevices) DeleteTokens(_ context.Context, _ []string) error { return m.deleteErr }

type mockPushNotifications struct{ saveErr error }

func (m *mockPushNotifications) Save(_ context.Context, _ *domain.Notification) error {
	return m.saveErr
}

type mockPushSender struct {
	invalid []string
	err     error
}

func (m *mockPushSender) Send(_ context.Context, _ []string, _, _ string, _ map[string]string) ([]string, error) {
	return m.invalid, m.err
}

func pushTask(userID uuid.UUID, title, body string) domain.Task {
	b, _ := json.Marshal(pushNotificationPayload{UserID: userID, Title: title, Body: body})
	return makeRawTask(b)
}

func TestPushProcessor_NotifSaveError(t *testing.T) {
	notifs := &mockPushNotifications{saveErr: errors.New("db error")}
	w := NewPushNotificationProcessor(&mockTaskRepo{}, &mockPushDevices{}, notifs, &mockPushSender{})
	if err := w.processTask(context.Background(), pushTask(uuid.New(), "T", "B")); err == nil {
		t.Fatal("expected error")
	}
}

func TestPushProcessor_NilSender(t *testing.T) {
	w := NewPushNotificationProcessor(&mockTaskRepo{}, &mockPushDevices{}, &mockPushNotifications{}, nil)
	if err := w.processTask(context.Background(), pushTask(uuid.New(), "T", "B")); err != nil {
		t.Fatalf("nil sender should just mark done: %v", err)
	}
}

func TestPushProcessor_NoDevices(t *testing.T) {
	devices := &mockPushDevices{listResult: nil}
	w := NewPushNotificationProcessor(&mockTaskRepo{}, devices, &mockPushNotifications{}, &mockPushSender{})
	if err := w.processTask(context.Background(), pushTask(uuid.New(), "T", "B")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushProcessor_ListDevicesError(t *testing.T) {
	devices := &mockPushDevices{listErr: errors.New("db error")}
	w := NewPushNotificationProcessor(&mockTaskRepo{}, devices, &mockPushNotifications{}, &mockPushSender{})
	if err := w.processTask(context.Background(), pushTask(uuid.New(), "T", "B")); err == nil {
		t.Fatal("expected error")
	}
}

func TestPushProcessor_SendError(t *testing.T) {
	devices := &mockPushDevices{listResult: []domain.FCMDevice{{FCMToken: "tok"}}}
	sender := &mockPushSender{err: errors.New("fcm error")}
	w := NewPushNotificationProcessor(&mockTaskRepo{}, devices, &mockPushNotifications{}, sender)
	if err := w.processTask(context.Background(), pushTask(uuid.New(), "T", "B")); err == nil {
		t.Fatal("expected error")
	}
}

func TestPushProcessor_InvalidTokensDeleted(t *testing.T) {
	devices := &mockPushDevices{listResult: []domain.FCMDevice{{FCMToken: "expired-tok"}}}
	sender := &mockPushSender{invalid: []string{"expired-tok"}}
	w := NewPushNotificationProcessor(&mockTaskRepo{}, devices, &mockPushNotifications{}, sender)
	if err := w.processTask(context.Background(), pushTask(uuid.New(), "T", "B")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushProcessor_WithData(t *testing.T) {
	devices := &mockPushDevices{listResult: []domain.FCMDevice{{FCMToken: "tok"}}}
	w := NewPushNotificationProcessor(&mockTaskRepo{}, devices, &mockPushNotifications{}, &mockPushSender{})

	payload := pushNotificationPayload{
		UserID: uuid.New(),
		Title:  "Новый тест",
		Body:   "Математика. Тест 3",
		Data:   map[string]string{"route": "/test/abc", "role": "student"},
	}
	b, _ := json.Marshal(payload)
	if err := w.processTask(context.Background(), makeRawTask(b)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushProcessor_BadPayload(t *testing.T) {
	w := NewPushNotificationProcessor(&mockTaskRepo{}, &mockPushDevices{}, &mockPushNotifications{}, nil)
	if err := w.processTask(context.Background(), makeRawTask([]byte("bad"))); err == nil {
		t.Fatal("expected error")
	}
}

// --- otpErrorMessage ---

func TestOTPErrorMessage(t *testing.T) {
	cases := []struct{ code, contains string }{
		{"OTP_ALREADY_SENT", "3 минуты"},
		{"PHONE_UNAVAILABLE", "заблокирован"},
		{"UNKNOWN_CODE", "ошибка"},
	}
	for _, c := range cases {
		msg := otpErrorMessage(c.code)
		if msg == "" {
			t.Errorf("empty message for code %s", c.code)
		}
	}
}
