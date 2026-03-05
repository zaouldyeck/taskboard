package task

import (
	"context"
	"database/sql"
	"fmt"
)

// TaskDB implements Store interface so that
// we can store data about our tasks in our DB.
type TaskDB struct {
	db *sql.DB
}

// NewTaskDB instantation is needed for DB operations methods
// against this type.
func NewTaskDB(db *sql.DB) *TaskDB {
	return &TaskDB{db: db}
}

func (t *TaskDB) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (board_id, title, description, completed, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := t.db.QueryRowContext(
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

func (t *TaskDB) GetByID(ctx context.Context, id int64) (*Task, error) {
	query := `
		SELECT id, board_id, title, description, completed, created_by, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	task := &Task{}
	err := t.db.QueryRowContext(ctx, query, id).Scan(
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
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

func (t *TaskDB) List(ctx context.Context, boardID int64, completed *bool, limit, offset int) ([]*Task,
	int, error,
) {
	query := `
		SELECT id, board_id, title, description, completed, created_by, created_at, updated_at
		FROM tasks
		WHERE board_id = $1
	`

	params := []any{boardID}
	paramCount := 1

	// User specified a filter.
	if completed != nil {
		paramCount++
		query += fmt.Sprintf(" AND completed = $%d", paramCount)
		params = append(params, *completed)
	}

	query += " ORDER BY created_at DESC"

	paramCount++
	query += fmt.Sprintf(" LIMIT $%d", paramCount)
	params = append(params, limit)

	paramCount++
	query += fmt.Sprintf(" OFFSET $%d", paramCount)
	params = append(params, offset)

	rows, err := t.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

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

	countQuery := `SELECT COUNT(*) FROM tasks WHERE board_id = $1`
	countParams := []any{boardID}
	// User specified a filter.
	if completed != nil {
		countQuery += " AND completed = $2"
		countParams = append(countParams, *completed)
	}

	var totalCount int
	err = t.db.QueryRowContext(ctx, countQuery, countParams...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return tasks, totalCount, nil
}

func (t *TaskDB) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks
		SET title = $1, description = $2, completed = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`

	err := t.db.QueryRowContext(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Completed,
		task.ID,
	).Scan(&task.UpdatedAt)

	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

func (t *TaskDB) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := t.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
