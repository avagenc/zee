package account

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Account struct {
	OwnerID   string
	TuyaUID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *repository {
	return &repository{pool: pool}
}

func (r *repository) GetTuyaUID(ctx context.Context, ownerID string) (string, error) {
	var tuyaUID string
	query := `SELECT tuya_uid FROM tuya_app_accounts WHERE owner_id = $1 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, query, ownerID).Scan(&tuyaUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotLinked
		}
		return "", err
	}

	return tuyaUID, nil
}

func (r *repository) Get(ctx context.Context, ownerID string) (Account, error) {
	var acc Account
	query := `SELECT owner_id, tuya_uid, created_at, updated_at FROM tuya_app_accounts WHERE owner_id = $1 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, query, ownerID).Scan(&acc.OwnerID, &acc.TuyaUID, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Account{}, errors.New("tuya account not found")
		}
		return Account{}, err
	}

	return acc, nil
}
