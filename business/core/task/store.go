package task

import "context"

// Store defines the type of ways we interact with our DB
// for tasks.
type Store interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	List(ctx context.Context, boardID int64, completed *bool, limit, offset int) ([]*Task, int, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int64) error
}
