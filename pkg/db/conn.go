package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jensilo/harmony/pkg/types"
)

func NewConn(ctx context.Context, conn *types.PgCfg) (*pgxpool.Pool, error) {
	db, err := pgxpool.NewWithConfig(ctx, conn.Cfg)
	if err != nil {
		return nil, fmt.Errorf("can not set up pg conntection pool: %w", err)
	}

	return db, nil
}
