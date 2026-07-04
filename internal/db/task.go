package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID         string    `json:"id"`
	BoardID    string    `json:"board_id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	AssigneeID *string   `json:"assignee_id,omitempty"`
	Position   float64   `json:"position"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

func (r *TaskRepo) Create(ctx context.Context, t *Task) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO tasks (board_id, title, status, assignee_id, position)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, version, created_at, updated_at`,
		t.BoardID, t.Title, t.Status, t.AssigneeID, t.Position,
	).Scan(&t.ID, &t.Version, &t.CreatedAt, &t.UpdatedAt)
}

func (r *TaskRepo) GetByID(ctx context.Context, id string) (*Task, error) {
	t := &Task{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, board_id, title, status, assignee_id, position, version, created_at, updated_at
		 FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.BoardID, &t.Title, &t.Status, &t.AssigneeID, &t.Position,
		&t.Version, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (r *TaskRepo) GetByBoardID(ctx context.Context, boardID string) ([]*Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, board_id, title, status, assignee_id, position, version, created_at, updated_at
		 FROM tasks WHERE board_id = $1 ORDER BY position`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		if err := rows.Scan(&t.ID, &t.BoardID, &t.Title, &t.Status, &t.AssigneeID,
			&t.Position, &t.Version, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *TaskRepo) UpdateWithVersion(ctx context.Context, t *Task, expectedVersion int) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks
		 SET title = $1, status = $2, assignee_id = $3, position = $4,
		     version = version + 1, updated_at = now()
		 WHERE id = $5 AND version = $6`,
		t.Title, t.Status, t.AssigneeID, t.Position, t.ID, expectedVersion,
	)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		t.Version = expectedVersion + 1
		t.UpdatedAt = time.Now().UTC()
		return true, nil
	}
	return false, nil
}
