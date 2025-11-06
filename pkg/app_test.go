package pkg

import (
	"context"
	"os"
	"testing"

	"github.com/jensilo/cognireq/pkg/cfg"
)

func TestRun(t *testing.T) {
	// due to gemini right now not working - fix with mock LLM
	//cancel := runApp(t)
	//cancel()
}

func runApp(t *testing.T) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	t.Run("app runner", func(t *testing.T) {
		t.Parallel()
		if err := Run(ctx, cfg.DefaultGetEnv, os.Stdout, os.Stderr, os.Interrupt); err != nil {
			t.Errorf("%s\n", err)
		}
	})

	return cancel
}
