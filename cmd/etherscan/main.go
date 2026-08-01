package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"

	"github.com/etherscan/etherscan-cli/internal/cli"
)

// These are overridden at release time via -ldflags (see .goreleaser.yaml). A
// plain `go install`/`go build` leaves them at these defaults, so buildInfo()
// falls back to the module build info the toolchain embeds.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	root := cli.NewRootCommand(buildInfo())
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// buildInfo resolves version/commit/date. GoReleaser's ldflags win when present;
// otherwise (e.g. `go install github.com/etherscan/etherscan-cli/cmd/etherscan@latest`)
// it reads the version the Go toolchain stamps into the module build info, so
// `etherscan version` reports the installed tag/pseudo-version instead of "dev".
func buildInfo() cli.BuildInfo {
	v, c, d := version, commit, date
	if v == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
				v = strings.TrimPrefix(mv, "v")
			}
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if c == "none" {
						c = s.Value
					}
				case "vcs.time":
					if d == "unknown" {
						d = s.Value
					}
				}
			}
		}
	}
	return cli.BuildInfo{Version: v, Commit: c, Date: d}
}
