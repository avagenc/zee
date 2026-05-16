package zeedb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ValidateSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var exists bool
	query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'tuya_app_accounts')"

	err := pool.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check schema validation: %w", err)
	}

	if !exists {
		return fmt.Errorf("critical table 'tuya_app_accounts' does not exist. Please run database migrations")
	}

	return nil
}
