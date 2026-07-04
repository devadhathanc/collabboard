package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/devadhathan/collabboard/internal/auth"
)

type RefreshTokenStore struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenStore(pool *pgxpool.Pool) *RefreshTokenStore {
	return &RefreshTokenStore{pool: pool}
}

func (s *RefreshTokenStore) FindByHash(ctx context.Context, hash string) (*auth.RefreshToken, error) {
	rt := &auth.RefreshToken{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, family_id, revoked_at
		 FROM refresh_tokens WHERE token_hash = $1`, hash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.FamilyID, &rt.RevokedAt)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (s *RefreshTokenStore) Create(ctx context.Context, userID, tokenHash, familyID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, family_id) VALUES ($1, $2, $3)`,
		userID, tokenHash, familyID,
	)
	return err
}

func (s *RefreshTokenStore) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE id = $2`, now, id)
	return err
}

func (s *RefreshTokenStore) RevokeFamily(ctx context.Context, familyID string) error {
	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`,
		now, familyID)
	return err
}
