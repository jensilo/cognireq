package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConn(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("can not set up pg conntection pool: %w", err)
	}

	return db, nil
}

func IsUniqueConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}

func IntoUUID(pgUUID pgtype.UUID) (uuid.UUID, error) {
	id, err := uuid.Parse(pgUUID.String())
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func MustIntoUUID(pgUUID pgtype.UUID) uuid.UUID {
	id, err := IntoUUID(pgUUID)
	if err != nil {
		panic(err)
	}

	return id
}
