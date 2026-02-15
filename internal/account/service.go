package account

import (
	"context"

	"github.com/avagenc/zee-api/internal/domain"
)

var ErrNotLinked = domain.ErrAccountNotLinked

type Repository interface {
	Get(ctx context.Context, ownerID string) (Account, error)
	GetTuyaUID(ctx context.Context, ownerID string) (string, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) Get(ctx context.Context, ownerID string) (Account, error) {
	return s.repo.Get(ctx, ownerID)
}

func (s *service) GetTuyaUID(ctx context.Context, ownerID string) (string, error) {
	return s.repo.GetTuyaUID(ctx, ownerID)
}
