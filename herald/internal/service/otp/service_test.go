package otpsvc

import (
	"context"
	"errors"
	"testing"

	"herald/internal/domain"
)

type mockTxManager struct{ err error }

func (m *mockTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(ctx)
}

type mockTaskRepo struct{ scheduleErr error }

func (m *mockTaskRepo) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.scheduleErr
}

type mockPendingOTPRepo struct {
	saveErr   error
	getResult *domain.PendingOTP
	getErr    error
	deleteErr error
}

func (m *mockPendingOTPRepo) Save(_ context.Context, _ string, _ domain.Channel, _ int64) error {
	return m.saveErr
}
func (m *mockPendingOTPRepo) Get(_ context.Context, _ string, _ domain.Channel) (*domain.PendingOTP, error) {
	return m.getResult, m.getErr
}
func (m *mockPendingOTPRepo) Delete(_ context.Context, _ string, _ domain.Channel) error {
	return m.deleteErr
}

func TestRequestOTP_Success(t *testing.T) {
	svc := NewService(&mockTxManager{}, &mockTaskRepo{}, &mockPendingOTPRepo{})
	if err := svc.RequestOTP(context.Background(), 123, "+79001234567", domain.ChannelTG); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestOTP_PendingOTPSaveError(t *testing.T) {
	svc := NewService(&mockTxManager{}, &mockTaskRepo{}, &mockPendingOTPRepo{saveErr: errors.New("db error")})
	if err := svc.RequestOTP(context.Background(), 123, "+79001234567", domain.ChannelTG); err == nil {
		t.Fatal("expected error")
	}
}

func TestRequestOTP_ScheduleError(t *testing.T) {
	svc := NewService(&mockTxManager{}, &mockTaskRepo{scheduleErr: errors.New("db error")}, &mockPendingOTPRepo{})
	if err := svc.RequestOTP(context.Background(), 123, "+79001234567", domain.ChannelTG); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPendingOTP_Found(t *testing.T) {
	expected := &domain.PendingOTP{Phone: "+79001234567", ChatID: 123}
	svc := NewService(&mockTxManager{}, &mockTaskRepo{}, &mockPendingOTPRepo{getResult: expected})

	result, err := svc.GetPendingOTP(context.Background(), "+79001234567", domain.ChannelTG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Fatal("unexpected result")
	}
}

func TestGetPendingOTP_Error(t *testing.T) {
	svc := NewService(&mockTxManager{}, &mockTaskRepo{}, &mockPendingOTPRepo{getErr: errors.New("db error")})
	_, err := svc.GetPendingOTP(context.Background(), "+79001234567", domain.ChannelTG)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeletePendingOTP_Success(t *testing.T) {
	svc := NewService(&mockTxManager{}, &mockTaskRepo{}, &mockPendingOTPRepo{})
	if err := svc.DeletePendingOTP(context.Background(), "+79001234567", domain.ChannelTG); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePendingOTP_Error(t *testing.T) {
	svc := NewService(&mockTxManager{}, &mockTaskRepo{}, &mockPendingOTPRepo{deleteErr: errors.New("db error")})
	if err := svc.DeletePendingOTP(context.Background(), "+79001234567", domain.ChannelTG); err == nil {
		t.Fatal("expected error")
	}
}
