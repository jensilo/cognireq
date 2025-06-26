package cfg

import (
	"fmt"
	"github.com/jensilo/harmony/pkg"
	"github.com/jensilo/harmony/pkg/types"
	"os"
)

type App struct {
	// Pg set by "PG_DATABASE_URL" env var. Must be a valid PostgreSQL connection string.
	Pg *types.PgCfg
}

type Server struct {
	BasePath string
	Port     int
}

func AppCfgFromEnv(log pkg.Log) (*App, error) {
	// see `App.Pg`
	pgUrl := os.Getenv("PG_DATABASE_URL")
	if pgUrl == "" {
		log.Info("PG_DATABASE_URL: no pg connection string provided, using default")
		pgUrl = "user=user password=password host=db port=5432 dbname=harmony"
	}

	pgCfg, err := types.NewPgCfg(pgUrl)
	if err != nil {
		return nil, fmt.Errorf("error constructing app config: %w", err)
	}

	// ... more app cfg here

	return &App{Pg: pgCfg}, nil
}
