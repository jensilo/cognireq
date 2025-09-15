package test

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jensilo/harmony/pkg/db"
)

//go:embed seed.sql
var seed []byte

const DBConnStr = "user=user password=password host=localhost port=6543 dbname=harmony"

func InitDB(t *testing.T) *pgxpool.Pool {
	schemaID := sanitizeSchemaName(fmt.Sprintf("test_%s", uuid.New().String()))
	dbConnStr := fmt.Sprintf("%s search_path=%s", DBConnStr, schemaID)

	dbConn, err := db.NewConn(t.Context(), dbConnStr)
	if err != nil {
		t.Fatalf("failed to connect to db (%s): %v", DBConnStr, err)
	}

	if err := dbConn.Ping(t.Context()); err != nil {
		t.Fatalf("IMPORTANT: is dev docker container `testdb` started? can not ping db: %v", err)
	}

	if _, err := dbConn.Exec(t.Context(), fmt.Sprintf("CREATE SCHEMA %s", schemaID)); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	if _, err := dbConn.Exec(t.Context(), string(seed)); err != nil {
		t.Fatalf("failed to seed test db: %v", err)
	}

	t.Cleanup(func() {
		// fix t.Context() cancelled bug: use a new context for cleanup to ensure it runs to completion
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := dbConn.Exec(cleanupCtx, fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaID)); err != nil {
			t.Errorf("failed to drop schema: %v", err)
		}
		dbConn.Close()
	})

	return dbConn
}

func sanitizeSchemaName(schema string) string {
	schema = strings.ReplaceAll(schema, "-", "")
	return pgx.Identifier{schema}.Sanitize()
}

func InitLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
