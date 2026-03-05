package task

import "time"

// Task defines a task object which gives us a way to store specific
// details about a specific task and store that in our DB.
type Task struct {
	ID          int64
	BoardID     int64
	Title       string
	Description string
	Completed   bool
	CreatedBy   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
