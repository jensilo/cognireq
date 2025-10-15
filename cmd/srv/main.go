package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jensilo/cognireq/pkg"
	"github.com/jensilo/cognireq/pkg/cfg"
)

func main() {
	ctx := context.Background()
	if err := pkg.Run(ctx, cfg.DefaultGetEnv, os.Stdout, os.Stderr, os.Interrupt); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
