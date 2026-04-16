package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

const (
	maxOccurrences = 30
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:          normalized.Title,
		Description:    normalized.Description,
		Status:         normalized.Status,
		RecurrenceType: taskdomain.RecurrenceType(input.RecurrenceType),
	}

	jsonBytes, err := json.Marshal(input.RecurrenceData)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid recurrence data", ErrInvalidInput)
	}
	model.RecurrenceData = string(jsonBytes)

	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	tasks := s.generateTasks(model)

	var lastCreated *taskdomain.Task
	for _, t := range tasks {
		created, err := s.repo.Create(ctx, &t)
		if err != nil {
			return nil, err
		}
		lastCreated = created
	}

	return lastCreated, nil
}

func (s *Service) generateTasks(base *taskdomain.Task) []taskdomain.Task {
	switch base.RecurrenceType {
	case taskdomain.RecurrenceDaily:
		return s.generateDaily(base)
	case taskdomain.RecurrenceMonthly:
		return s.generateMonthly(base)
	case taskdomain.RecurrenceSpecific:
		return s.generateSpecific(base)
	case taskdomain.RecurrenceOddEven:
		return s.generateOddEven(base)
	default:
		return []taskdomain.Task{*base}
	}
}

func (s *Service) generateDaily(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task

	var data struct {
		Interval int `json:"interval"`
	}

	_ = json.Unmarshal([]byte(base.RecurrenceData), &data)

	if data.Interval <= 0 {
		data.Interval = 1
	}

	for i := 0; i < maxOccurrences; i++ {
		t := *base
		t.CreatedAt = base.CreatedAt.AddDate(0, 0, i*data.Interval)
		t.UpdatedAt = t.CreatedAt

		result = append(result, t)
	}

	return result
}

func (s *Service) generateMonthly(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task

	var data struct {
		Days []int `json:"days"`
	}

	_ = json.Unmarshal([]byte(base.RecurrenceData), &data)

	current := base.CreatedAt

	for i := 0; i < maxOccurrences; i++ {
		for _, day := range data.Days {
			if day < 1 || day > 31 {
				continue
			}

			date := time.Date(current.Year(), current.Month(), day, 0, 0, 0, 0, time.UTC)

			t := *base
			t.CreatedAt = date
			t.UpdatedAt = date

			result = append(result, t)
		}

		current = current.AddDate(0, 1, 0)
	}

	return result
}

func (s *Service) generateSpecific(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task

	var data struct {
		Dates []string `json:"dates"`
	}

	_ = json.Unmarshal([]byte(base.RecurrenceData), &data)

	for _, d := range data.Dates {
		parsed, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}

		t := *base
		t.CreatedAt = parsed
		t.UpdatedAt = parsed

		result = append(result, t)
	}

	return result
}

func (s *Service) generateOddEven(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task

	var data struct {
		Type string `json:"type"`
	}

	_ = json.Unmarshal([]byte(base.RecurrenceData), &data)

	current := base.CreatedAt

	for i := 0; i < maxOccurrences; i++ {
		if (current.Day()%2 == 0 && data.Type == "even") ||
			(current.Day()%2 != 0 && data.Type == "odd") {

			t := *base
			t.CreatedAt = current
			t.UpdatedAt = current

			result = append(result, t)
		}

		current = current.AddDate(0, 0, 1)
	}

	return result
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}
