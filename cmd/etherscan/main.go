package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/etherscan/etherscan-cli/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	root := cli.NewRootCommand(cli.BuildInfo{Version: version, Commit: commit, Date: date})
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
