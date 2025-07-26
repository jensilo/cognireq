package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jensilo/harmony/pkg"
	"github.com/jensilo/harmony/pkg/cfg"
	"github.com/jensilo/harmony/pkg/db"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, cfg.DefaultGetEnv, os.Stdout, os.Stderr, os.Interrupt); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	getenv func(string) string,
	stdout, stderr io.Writer,
	interrupt os.Signal,
) error {
	ctx, cancel := signal.NotifyContext(ctx, interrupt, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(stdout, nil))
	webCfg := cfg.WebConfigFromEnv(getenv)

	dbConn, err := db.NewConn(ctx, getenv(cfg.KeyDatabaseUrl))
	if err != nil {
		return fmt.Errorf("can not initialize db conn: %w", err)
	}
	defer dbConn.Close()
	queries := db.New(dbConn)

	srv, err := pkg.NewServer(logger)
	if err != nil {
		return fmt.Errorf("error constructing server mux: %s", err)
	}
	webServer := &http.Server{Addr: net.JoinHostPort(webCfg.Host, webCfg.Port), Handler: srv}

	go func() {
		logger.Info("web server listening", "address", webServer.Addr)
		if err := webServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_, _ = fmt.Fprintf(stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		logger.Info("shutting down gracefully (max. 10 seconds)...")
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, time.Second*10)
		defer cancel()

		if err := webServer.Shutdown(shutdownCtx); err != nil {
			_, _ = fmt.Fprintf(stderr, "error shutting down http server: %s\n", err)
		}
	}()
	wg.Wait()

	usr, err := queries.UserByEmail(ctx, "mail@jensheise.com") // remove
	logger.Info(fmt.Sprintf("user: %s, %s", usr.LastName, usr.FirstName))

	return nil
}
