package types

import "github.com/jackc/pgx/v5/pgxpool"

type PgCfg struct {
	Cfg *pgxpool.Config
}

func NewPgCfg(url string) (*PgCfg, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}

	return &PgCfg{Cfg: cfg}, nil
}
