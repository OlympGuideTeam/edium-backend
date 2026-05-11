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

type mockScheduler struct {
	calls []domain.TaskType
	err   error
}

func (m *mockScheduler) Schedule(_ context.Context, tt domain.TaskType, _ []byte) error {
	m.calls = append(m.calls, tt)
	return m.err
}

type mockDeviceDeleter struct{ err error }

func (m *mockDeviceDeleter) DeleteByUserID(_ context.Context, _ uuid.UUID) error { return m.err }

func marshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// --- AttemptScoredConsumer ---

func TestAttemptScored_LLM_WithAuthor(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &AttemptScoredConsumer{tasks: scheduler}

	msg := attemptScoredMsg{
		AttemptID: uuid.New(),
		SessionID: uuid.New(),
		AuthorID:  uuid.New(),
		GradedBy:  "llm",
	}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 1 || scheduler.calls[0] != domain.PushNotification {
		t.Errorf("expected PushNotification scheduled, got %v", scheduler.calls)
	}
}

func TestAttemptScored_LLM_NilAuthor(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &AttemptScoredConsumer{tasks: scheduler}

	msg := attemptScoredMsg{GradedBy: "llm", AuthorID: uuid.Nil}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 0 {
		t.Error("should not schedule when AuthorID is nil")
	}
}

func TestAttemptScored_Student_WithUser(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &AttemptScoredConsumer{tasks: scheduler}

	msg := attemptScoredMsg{
		SessionID:  uuid.New(),
		UserID:     uuid.New(),
		TotalScore: 8.5,
		MaxScore:   10,
		GradedBy:   "teacher",
	}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 1 || scheduler.calls[0] != domain.PushNotification {
		t.Errorf("expected PushNotification scheduled, got %v", scheduler.calls)
	}
}

func TestAttemptScored_Student_NilUser(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &AttemptScoredConsumer{tasks: scheduler}

	msg := attemptScoredMsg{GradedBy: "teacher", UserID: uuid.Nil}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 0 {
		t.Error("should not schedule when UserID is nil")
	}
}

func TestAttemptScored_BadJSON(t *testing.T) {
	c := &AttemptScoredConsumer{tasks: &mockScheduler{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- CourseSessionNotifyConsumer ---

func TestCourseSessionNotify_MultipleUsers(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &CourseSessionNotifyConsumer{tasks: scheduler}

	msg := courseSessionNotifyMsg{
		SessionID: uuid.New().String(),
		UserIDs:   []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		Title:     "Алгебра. Тест 1",
	}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 3 {
		t.Errorf("expected 3 scheduled tasks, got %d", len(scheduler.calls))
	}
}

func TestCourseSessionNotify_EmptyUsers(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &CourseSessionNotifyConsumer{tasks: scheduler}

	msg := courseSessionNotifyMsg{SessionID: uuid.New().String(), UserIDs: nil}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 0 {
		t.Errorf("expected 0 scheduled tasks, got %d", len(scheduler.calls))
	}
}

func TestCourseSessionNotify_ScheduleError_Continues(t *testing.T) {
	scheduler := &mockScheduler{err: errors.New("db error")}
	c := &CourseSessionNotifyConsumer{tasks: scheduler}

	msg := courseSessionNotifyMsg{
		SessionID: uuid.New().String(),
		UserIDs:   []uuid.UUID{uuid.New(), uuid.New()},
	}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("schedule errors should be logged, not returned: %v", err)
	}
}

func TestCourseSessionNotify_BadJSON(t *testing.T) {
	c := &CourseSessionNotifyConsumer{tasks: &mockScheduler{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- UserLogoutConsumer ---

func TestUserLogout_Success(t *testing.T) {
	devices := &mockDeviceDeleter{}
	c := &UserLogoutConsumer{devices: devices}

	msg := userLogoutMsg{UserID: uuid.New()}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserLogout_DeleteError(t *testing.T) {
	devices := &mockDeviceDeleter{err: errors.New("db error")}
	c := &UserLogoutConsumer{devices: devices}

	msg := userLogoutMsg{UserID: uuid.New()}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err == nil {
		t.Fatal("expected error")
	}
}

func TestUserLogout_BadJSON(t *testing.T) {
	c := &UserLogoutConsumer{devices: &mockDeviceDeleter{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- OTPSentConsumer ---

func TestOTPSentConsumer_Success(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &OTPSentConsumer{tasks: scheduler}

	msg := otpSentMsg{Phone: "+79001234567", OTP: 123456}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 1 || scheduler.calls[0] != domain.OTPSent {
		t.Errorf("expected OTPSent scheduled, got %v", scheduler.calls)
	}
}

func TestOTPSentConsumer_BadJSON(t *testing.T) {
	c := &OTPSentConsumer{tasks: &mockScheduler{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- QuizGenerationNotifyConsumer ---

func TestQuizGenerationNotify_NilAuthor(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &QuizGenerationNotifyConsumer{tasks: scheduler}

	msg := quizGenerationNotifyMsg{QuizID: uuid.New(), AuthorID: uuid.Nil, Title: "Тест"}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 0 {
		t.Error("should not schedule when AuthorID is nil")
	}
}

func TestQuizGenerationNotify_Success(t *testing.T) {
	scheduler := &mockScheduler{}
	c := &QuizGenerationNotifyConsumer{tasks: scheduler}

	msg := quizGenerationNotifyMsg{QuizID: uuid.New(), AuthorID: uuid.New(), Title: "Алгебра"}
	if err := c.handle(context.Background(), marshalJSON(t, msg)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scheduler.calls) != 1 || scheduler.calls[0] != domain.PushNotification {
		t.Errorf("expected PushNotification scheduled, got %v", scheduler.calls)
	}
}

func TestQuizGenerationNotify_BadJSON(t *testing.T) {
	c := &QuizGenerationNotifyConsumer{tasks: &mockScheduler{}}
	if err := c.handle(context.Background(), []byte("bad")); err == nil {
		t.Fatal("expected error")
	}
}

// --- helpers ---

func makeTask(payload any) domain.Task {
	b, _ := json.Marshal(payload)
	return domain.Task{ID: uuid.New(), Payload: b}
}

func makeRawTask(payload []byte) domain.Task {
	return domain.Task{ID: uuid.New(), Payload: payload, AvailableAt: time.Now()}
}
