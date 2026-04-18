package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

// maxOccurrences limits number of generated tasks
// to prevent uncontrolled data growth.
const (
	maxOccurrences = 30
)

// Service implements business logic for task management,
// including recurrence rules processing and task generation.
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
		RecurrenceType: taskdomain.RecurrenceType(normalized.RecurrenceType),
	}

	if normalized.RecurrenceType == "" {
		normalized.RecurrenceData = nil
	}

	jsonBytes, err := json.Marshal(normalized.RecurrenceData)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid recurrence data", ErrInvalidInput)
	}
	if input.RecurrenceData == nil {
		model.RecurrenceData = "{}"
	} else {
		model.RecurrenceData = string(jsonBytes)
	}

	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	if model.RecurrenceType == "" {
		return s.repo.Create(ctx, model)
	}

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

// generateTasks creates task instances based on recurrence rules.
// If no recurrence is specified, returns a single task.
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

// generateDaily creates tasks with a fixed day interval.
// Example: every 2 days.
func (s *Service) generateDaily(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task
	seen := make(map[string]bool)

	var data struct {
		Interval int `json:"interval"`
	}

	if err := json.Unmarshal([]byte(base.RecurrenceData), &data); err != nil {
		return []taskdomain.Task{*base}
	}

	if data.Interval <= 0 {
		data.Interval = 1
	}

	for i := 0; i < maxOccurrences; i++ {
		t := *base
		t.CreatedAt = base.CreatedAt.AddDate(0, 0, i*data.Interval)
		t.UpdatedAt = t.CreatedAt

		key := t.CreatedAt.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true

		result = append(result, t)
	}

	return result
}

// generateMonthly creates tasks for specific days of the month.
// Invalid dates (e.g., Feb 30) are skipped.
func (s *Service) generateMonthly(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task
	seen := make(map[string]bool)

	var data struct {
		Days []int `json:"days"`
	}

	if err := json.Unmarshal([]byte(base.RecurrenceData), &data); err != nil {
		return []taskdomain.Task{*base}
	}

	current := base.CreatedAt

	for i := 0; i < maxOccurrences; i++ {
		for _, day := range data.Days {
			if day < 1 || day > 31 {
				continue
			}

			date := time.Date(current.Year(), current.Month(), day, 0, 0, 0, 0, time.UTC)

			if date.Month() != current.Month() {
				continue
			}
			t := *base
			t.CreatedAt = date
			t.UpdatedAt = date

			key := t.CreatedAt.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true

			result = append(result, t)
		}

		current = current.AddDate(0, 1, 0)
	}

	return result
}

// generateSpecific creates tasks for explicitly provided dates.
func (s *Service) generateSpecific(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task
	seen := make(map[string]bool)

	var data struct {
		Dates []string `json:"dates"`
	}

	if err := json.Unmarshal([]byte(base.RecurrenceData), &data); err != nil {
		return []taskdomain.Task{*base}
	}

	for _, d := range data.Dates {
		parsed, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}

		t := *base
		t.CreatedAt = parsed
		t.UpdatedAt = parsed

		key := t.CreatedAt.Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true

		result = append(result, t)
	}

	return result
}

// generateOddEven creates tasks only on odd or even days.
func (s *Service) generateOddEven(base *taskdomain.Task) []taskdomain.Task {
	var result []taskdomain.Task
	seen := make(map[string]bool)

	var data struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal([]byte(base.RecurrenceData), &data); err != nil {
		return []taskdomain.Task{*base}
	}

	if data.Type != "even" && data.Type != "odd" {
		return []taskdomain.Task{*base}
	}

	current := base.CreatedAt

	for i := 0; i < maxOccurrences; i++ {
		if (current.Day()%2 == 0 && data.Type == "even") ||
			(current.Day()%2 != 0 && data.Type == "odd") {

			t := *base
			t.CreatedAt = current
			t.UpdatedAt = current

			key := t.CreatedAt.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true

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

	if input.RecurrenceType != "" && input.RecurrenceData == nil {
		return CreateInput{}, fmt.Errorf("%w: recurrence data required", ErrInvalidInput)
	}

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.RecurrenceType != "" {
		if err := validateRecurrence(input.RecurrenceType, input.RecurrenceData); err != nil {
			return CreateInput{}, err
		}
	}

	return input, nil
}

func validateRecurrence(rType string, data any) error {
	if data == nil {
		return fmt.Errorf("%w: recurrence data required", ErrInvalidInput)
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("%w: invalid recurrence data format", ErrInvalidInput)
	}

	switch rType {

	case string(taskdomain.RecurrenceDaily):
		var d struct {
			Interval int `json:"interval"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("%w: invalid daily format", ErrInvalidInput)
		}
		if d.Interval <= 0 {
			return fmt.Errorf("%w: interval must be > 0", ErrInvalidInput)
		}

	case string(taskdomain.RecurrenceMonthly):
		var d struct {
			Days []int `json:"days"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("%w: invalid monthly format", ErrInvalidInput)
		}
		if len(d.Days) == 0 {
			return fmt.Errorf("%w: days required", ErrInvalidInput)
		}
		for _, day := range d.Days {
			if day < 1 || day > 31 {
				return fmt.Errorf("%w: day must be between 1 and 31", ErrInvalidInput)
			}
		}

	case string(taskdomain.RecurrenceSpecific):
		var d struct {
			Dates []string `json:"dates"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("%w: invalid specific format", ErrInvalidInput)
		}
		if len(d.Dates) == 0 {
			return fmt.Errorf("%w: dates required", ErrInvalidInput)
		}
		for _, date := range d.Dates {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return fmt.Errorf("%w: invalid date format (use YYYY-MM-DD)", ErrInvalidInput)
			}
		}

	case string(taskdomain.RecurrenceOddEven):
		var d struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("%w: invalid odd/even format", ErrInvalidInput)
		}
		if d.Type != "odd" && d.Type != "even" {
			return fmt.Errorf("%w: type must be 'odd' or 'even'", ErrInvalidInput)
		}

	default:
		return fmt.Errorf("%w: unknown recurrence type", ErrInvalidInput)
	}

	return nil
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
