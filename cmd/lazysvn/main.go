package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maiyangzhan/lazysvn/internal/svn"
	"github.com/maiyangzhan/lazysvn/internal/ui"
)

// version is injected at build time via -ldflags "-X main.version=..."
// Falls back to "dev" for unversioned builds (plain `go build`).
var version = "dev"

func main() {
	cwd := flag.String("cwd", ".", "working copy directory")
	logLimit := flag.Int("log-limit", 50, "number of log entries to show")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("lazysvn", version)
		return
	}

	client := svn.New(*cwd)
	app := ui.NewApp(client, *logLimit)

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazysvn: %v\n", err)
		os.Exit(1)
	}
}
