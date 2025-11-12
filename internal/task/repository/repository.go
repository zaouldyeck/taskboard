package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Task represents a task object in DB.
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

// Repository handles DB ops for tasks.
type Repository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	List(ctx context.Context, boardID int64, completed *bool, limit, offset int) ([]*Task, int, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id int64) error
}

type postgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (board_id, title, description, completed, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	// Validate that row could be added to table.
	err := r.db.QueryRowContext(
		ctx,
		query,
		task.BoardID,
		task.Title,
		task.Description,
		task.Completed,
		task.CreatedBy,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*Task, error) {
	query := `
		SELECT id, board_id, title, description, completed, created_by, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	task := &Task{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.BoardID,
		&task.Title,
		&task.Description,
		&task.Completed,
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Special case, for when no rows are found.
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task; %w", err)
	}

	return task, nil
}

// List lists tasks with the option to filter.
func (r *postgresRepository) List(ctx context.Context, boardID int64,
	completed *bool, limit, offset int,
) ([]*Task, int, error) {
	// Dynamic query based on filters.
	query := `
		SELECT id, board_id, title, description, completed, created_by, created_at, updated_at
		FROM tasks
		WHERE board_id = $1
	`

	// Track which parameter number we're on.
	params := []interface{}{boardID}
	paramCount := 1

	// Optional completed filter.
	if completed != nil {
		paramCount++
		query += fmt.Sprintf(" AND completed = $%d", paramCount)
		params = append(params, *completed)
	}

	query += " ORDER BY created_at DESC"

	// Set pagination.
	paramCount++
	query += fmt.Sprintf(" LIMIT $%d", paramCount)
	params = append(params, limit)

	paramCount++
	query += fmt.Sprintf(" OFFSET $%d", paramCount)
	params = append(params, offset)
	// --

	rows, err := r.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	// Scan DB rows into slice of Task ptrs.
	tasks := []*Task{}
	for rows.Next() {
		task := &Task{}
		err := rows.Scan(
			&task.ID,
			&task.BoardID,
			&task.Title,
			&task.Description,
			&task.Completed,
			&task.CreatedBy,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating tasks: %w", err)
	}

	// Get total count for pagination.
	countQuery := `SELECT COUNT(*) FROM tasks WHERE board_id = $1`
	countParams := []interface{}{boardID}
	if completed != nil {
		countQuery += " AND completed = $2"
		countParams = append(countParams, *completed)
	}

	var totalCount int
	err = r.db.QueryRowContext(ctx, countQuery, countParams...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return tasks, totalCount, nil
}

func (r *postgresRepository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks
		SET title = $1, description = $2, completed = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Completed,
		task.ID,
	).Scan(&task.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found")
	}
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// Delete task.
func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

type carParts struct {
	numWheels       int
	Make            string
	Model           string
	numOfSeats      int
	seatingMaterial string
	horsePower      string
	automatic       bool
}
