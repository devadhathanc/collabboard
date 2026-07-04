package search

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Result struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"board_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Rank      float64   `json:"rank"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) SearchTasks(ctx context.Context, boardID, query string, userID string) ([]Result, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.id, t.board_id, t.title, t.status,
		        ts_rank(to_tsvector('english', t.title), plainto_tsquery('english', $1)) AS rank,
		        t.updated_at
		 FROM tasks t
		 JOIN boards b ON b.id = t.board_id
		 JOIN memberships m ON m.workspace_id = b.workspace_id
		 WHERE t.board_id = $2
		   AND m.user_id = $3
		   AND to_tsvector('english', t.title) @@ plainto_tsquery('english', $1)
		 ORDER BY rank DESC
		 LIMIT 50`,
		query, boardID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.ID, &r.BoardID, &r.Title, &r.Status, &r.Rank, &r.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Service) ReindexTask(ctx context.Context, taskID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE tasks SET updated_at = now() WHERE id = $1`, taskID)
	return err
}
