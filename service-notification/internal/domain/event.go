package domain

// TaskCreatedEvent representa o payload task.created consumido.
type TaskCreatedEvent struct {
	TaskID int64  `json:"task_id"`
	Title  string `json:"title"`
}
