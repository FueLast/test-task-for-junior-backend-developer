package handlers

import (
	"encoding/json"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`

	RecurrenceType string          `json:"recurrence_type"`
	RecurrenceData json.RawMessage `json:"recurrence_data"`
}

type CreateTaskRequest struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`

	RecurrenceType string `json:"recurrence_type"`
	RecurrenceData any    `json:"recurrence_data"`
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	RecurrenceType taskdomain.RecurrenceType `json:"recurrence_type"`
	RecurrenceData json.RawMessage           `json:"recurrence_data"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	var raw json.RawMessage

	if len(task.RecurrenceData) > 0 {
		raw = json.RawMessage(task.RecurrenceData)
	}

	return taskDTO{
		ID:             task.ID,
		Title:          task.Title,
		Description:    task.Description,
		Status:         task.Status,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
		RecurrenceType: task.RecurrenceType,
		RecurrenceData: raw,
	}
}
