package domain

import "time"

type AsyncJob struct {
	ID             int64      `json:"id"`
	PlatformTaskID *int64     `json:"platform_task_id"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	Payload        string     `json:"payload"`
	Progress       float64    `json:"progress"`
	Error          *string    `json:"error"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
}
