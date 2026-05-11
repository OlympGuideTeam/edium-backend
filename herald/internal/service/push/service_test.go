package pushsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"herald/internal/domain"
)

type mockDevices struct {
	upsertErr       error
	deleteErr       error
	listResult      []domain.FCMDevice
	listErr         error
	deleteByUserErr error
	deleteTokensErr error
}

func (m *mockDevices) Upsert(_ context.Context, _ uuid.UUID, _, _ string) error {
	return m.upsertErr
}
func (m *mockDevices) Delete(_ context.Context, _ string) error { return m.deleteErr }
func (m *mockDevices) ListByUserID(_ context.Context, _ uuid.UUID) ([]domain.FCMDevice, error) {
	return m.listResult, m.listErr
}
func (m *mockDevices) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return m.deleteByUserErr
}
func (m *mockDevices) DeleteTokens(_ context.Context, _ []string) error { return m.deleteTokensErr }

type mockNotifications struct {
	saveErr        error
	listResult     []domain.Notification
	listErr        error
	markReadErr    error
	countUnreadVal int
	countUnreadErr error
}

func (m *mockNotifications) Save(_ context.Context, _ *domain.Notification) error { return m.saveErr }
func (m *mockNotifications) ListByUserID(_ context.Context, _ uuid.UUID) ([]domain.Notification, error) {
	return m.listResult, m.listErr
}
func (m *mockNotifications) MarkRead(_ context.Context, _, _ uuid.UUID) error { return m.markReadErr }
func (m *mockNotifications) CountUnread(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countUnreadVal, m.countUnreadErr
}

type mockSender struct {
	sendInvalid []string
	sendErr     error
	badgeErr    error
}

func (m *mockSender) Send(_ context.Context, _ []string, _, _ string, _ map[string]string, _ int) ([]string, error) {
	return m.sendInvalid, m.sendErr
}
func (m *mockSender) SendBadge(_ context.Context, _ []string, _ int) ([]string, error) {
	return m.sendInvalid, m.badgeErr
}

func TestRegisterDevice_Success(t *testing.T) {
	svc := NewService(&mockDevices{}, &mockNotifications{}, &mockSender{})
	if err := svc.RegisterDevice(context.Background(), uuid.New(), "token", "ios"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterDevice_Error(t *testing.T) {
	svc := NewService(&mockDevices{upsertErr: errors.New("db error")}, &mockNotifications{}, nil)
	if err := svc.RegisterDevice(context.Background(), uuid.New(), "token", "ios"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteDevice_Success(t *testing.T) {
	svc := NewService(&mockDevices{}, &mockNotifications{}, nil)
	if err := svc.DeleteDevice(context.Background(), "token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListNotifications_Success(t *testing.T) {
	notifs := []domain.Notification{{ID: uuid.New(), Title: "Test"}}
	svc := NewService(&mockDevices{}, &mockNotifications{listResult: notifs}, nil)

	result, err := svc.ListNotifications(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}
}

func TestMarkRead_MarkReadError(t *testing.T) {
	notifs := &mockNotifications{markReadErr: errors.New("db error")}
	svc := NewService(&mockDevices{}, notifs, &mockSender{})
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err == nil {
		t.Fatal("expected error")
	}
}

func TestMarkRead_NilSender(t *testing.T) {
	svc := NewService(&mockDevices{}, &mockNotifications{}, nil)
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkRead_CountUnreadError_LogsOnly(t *testing.T) {
	notifs := &mockNotifications{countUnreadErr: errors.New("db error")}
	svc := NewService(&mockDevices{}, notifs, &mockSender{})
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("count error should be logged, not returned: %v", err)
	}
}

func TestMarkRead_ListDevicesError_LogsOnly(t *testing.T) {
	devices := &mockDevices{listErr: errors.New("db error")}
	svc := NewService(devices, &mockNotifications{}, &mockSender{})
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("list devices error should be logged, not returned: %v", err)
	}
}

func TestMarkRead_NoDevices(t *testing.T) {
	svc := NewService(&mockDevices{listResult: nil}, &mockNotifications{}, &mockSender{})
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkRead_SendBadgeError_LogsOnly(t *testing.T) {
	devices := &mockDevices{listResult: []domain.FCMDevice{{FCMToken: "tok"}}}
	sender := &mockSender{badgeErr: errors.New("fcm error")}
	svc := NewService(devices, &mockNotifications{}, sender)
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("send badge error should be logged, not returned: %v", err)
	}
}

func TestMarkRead_InvalidTokensDeleted(t *testing.T) {
	devices := &mockDevices{listResult: []domain.FCMDevice{{FCMToken: "invalid-tok"}}}
	sender := &mockSender{sendInvalid: []string{"invalid-tok"}}
	svc := NewService(devices, &mockNotifications{}, sender)
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkRead_Success(t *testing.T) {
	devices := &mockDevices{listResult: []domain.FCMDevice{{FCMToken: "tok"}}}
	svc := NewService(devices, &mockNotifications{countUnreadVal: 3}, &mockSender{})
	if err := svc.MarkRead(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUserDevices_Success(t *testing.T) {
	svc := NewService(&mockDevices{}, &mockNotifications{}, nil)
	if err := svc.DeleteUserDevices(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
