package task

import (
	"context"
	"fmt"
)

// Service handles business logic for handling tasks.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create task in order to store it in the DB.
func (s *Service) Create(ctx context.Context, boardID int64, title, description string,
	createdBy int64,
) (*Task, error) {
	if boardID == 0 {
		return nil, ErrInvalidBoardID
	}
	if title == "" {
		return nil, ErrInvalidTitle
	}

	task := &Task{
		BoardID:     boardID,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedBy:   createdBy,
	}

	if err := s.store.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	return task, nil
}

// GetByID is used to retrieve a specific task from DB, used for listing tasks.
func (s *Service) GetByID(ctx context.Context, taskID int64) (*Task, error) {
	if taskID == 0 {
		return nil, ErrInvalidTaskID
	}

	task, err := s.store.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("retrieving task: %w", err)
	}

	return task, nil
}

// List is used for viewing tasks of a board.
func (s *Service) List(ctx context.Context, boardID int64, completed *bool, pageSize,
	pageNumber int32,
) ([]*Task, int, error) {
	if boardID == 0 {
		return nil, 0, ErrInvalidBoardID
	}

	// Setup of default pagination of the taskboard.
	if pageSize == 0 {
		pageSize = 50 // Default page size.
	}
	if pageSize > 100 {
		pageSize = 100 // Max page size.
	}

	if pageNumber < 1 {
		pageNumber = 1
	}

	// Calculate offset from page number.
	offset := (pageNumber - 1) * pageSize

	// Call List on store to get the tasks for the specific board.
	tasks, totalCount, err := s.store.List(ctx, boardID, completed, int(pageSize), int(offset))
	if err != nil {
		return nil, 0, fmt.Errorf("listing tasks: %w", err)
	}

	return tasks, totalCount, nil
}

// Update handles updating a task with optional fields data.
func (s *Service) Update(ctx context.Context, taskID int64, title, description *string, completed *bool) (*Task, error) {
	if taskID == 0 {
		return nil, ErrInvalidTaskID
	}

	task, err := s.store.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("retrieving task: %w", err)
	}

	// Optional updates.
	if title != nil {
		task.Title = *title
	}
	if description != nil {
		task.Description = *description
	}
	if completed != nil {
		task.Completed = *completed
	}

	if err := s.store.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("updating task: %w", err)
	}

	return task, nil
}

// Delete handles deletion of tasks to have a way to purge tasks from DB if needed.
func (s *Service) Delete(ctx context.Context, taskID int64) error {
	if taskID == 0 {
		return ErrInvalidTaskID
	}

	if err := s.store.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}

	return nil
}
