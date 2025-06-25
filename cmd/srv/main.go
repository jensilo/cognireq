package main

import (
	"context"
	"fmt"
	"github.com/jensilo/harmony/pkg/cfg"
	"github.com/jensilo/harmony/pkg/db"
	"log/slog"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	appCfg, err := cfg.AppCfgFromEnv(log)
	if err != nil {
		panic(err)
	}

	dbConn, err := db.NewConn(ctx, appCfg.Pg)
	if err != nil {
		panic(err)
	}
	defer dbConn.Close()
	queries := db.New(dbConn)

	usr, err := queries.UserByEmail(ctx, "mail@jensheise.com")
	fmt.Printf("%v - %v", usr, err)

	fmt.Println("hello, world!")
}
