package domain

type TaskCreatedEvent struct {
	TaskID int64  `json:"task_id"`
	Title  string `json:"title"`
}
