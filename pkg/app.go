package pkg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jensilo/cognireq/pkg/cfg"
	"github.com/jensilo/cognireq/pkg/db"
)

type Services struct {
	User *UserService
	IGA  *IGAService
}

func NewServices(us *UserService, is *IGAService) *Services {
	return &Services{User: us, IGA: is}
}

func Run(
	ctx context.Context,
	getenv cfg.GetEnvFunc,
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

	services := NewServices(
		NewUserService(queries),
		NewIGAService(queries),
	)

	go StartSessionCleanup(ctx, logger, services.User, 90*time.Minute)

	mux, err := NewServer(logger, services)
	srv := AddAPIMiddlewares(getenv, mux)

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

	return nil
}
