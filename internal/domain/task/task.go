package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type RecurrenceType string

const (
	RecurrenceNone     RecurrenceType = ""
	RecurrenceDaily    RecurrenceType = "daily"
	RecurrenceMonthly  RecurrenceType = "monthly"
	RecurrenceSpecific RecurrenceType = "specific"
	RecurrenceOddEven  RecurrenceType = "odd_even"
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	RecurrenceType RecurrenceType `json:"recurrence_type"`
	RecurrenceData string         `json:"recurrence_data"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}
