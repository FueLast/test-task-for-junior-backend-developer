package task

import (
	"context"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type mockRepo struct{}

func (m *mockRepo) Create(ctx context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	t.ID = 1
	return t, nil
}
func (m *mockRepo) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	return nil, nil
}
func (m *mockRepo) Update(ctx context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	return t, nil
}
func (m *mockRepo) Delete(ctx context.Context, id int64) error {
	return nil
}
func (m *mockRepo) List(ctx context.Context) ([]taskdomain.Task, error) {
	return nil, nil
}

func TestCreate_InvalidDailyInterval(t *testing.T) {
	service := NewService(&mockRepo{})

	_, err := service.Create(context.Background(), CreateInput{
		Title:          "test",
		RecurrenceType: "daily",
		RecurrenceData: map[string]any{
			"interval": 0,
		},
	})

	if err == nil {
		t.Fatal("expected error for interval=0, got nil")
	}
}

func TestCreate_InvalidSpecificDate(t *testing.T) {
	service := NewService(&mockRepo{})

	_, err := service.Create(context.Background(), CreateInput{
		Title:          "test",
		RecurrenceType: "specific",
		RecurrenceData: map[string]any{
			"dates": []string{"2026-02-30"},
		},
	})

	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}

func TestGenerateMonthly_SkipInvalidDates(t *testing.T) {
	service := NewService(&mockRepo{})

	base := &taskdomain.Task{
		Title:          "test",
		Status:         taskdomain.StatusNew,
		CreatedAt:      time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Now(),
		RecurrenceType: taskdomain.RecurrenceMonthly,
		RecurrenceData: `{"days":[31]}`,
	}

	tasks := service.generateMonthly(base)

	for _, tsk := range tasks {
		if tsk.CreatedAt.Month() == 2 && tsk.CreatedAt.Day() == 31 {
			t.Fatal("generated invalid date Feb 31")
		}
	}
}
