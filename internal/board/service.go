package board

import (
	"context"

	"github.com/devadhathan/collabboard/internal/db"
)

type TaskStore interface {
	Create(ctx context.Context, t *db.Task) error
	GetByID(ctx context.Context, id string) (*db.Task, error)
	UpdateWithVersion(ctx context.Context, t *db.Task, expectedVersion int) (bool, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, channel string, msg []byte) error
}

type Service struct {
	store TaskStore
	pub   EventPublisher
}

func NewService(store TaskStore, pub EventPublisher) *Service {
	return &Service{store: store, pub: pub}
}
