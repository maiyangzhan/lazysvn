package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maiyangzhan/lazysvn/internal/svn"
	"github.com/maiyangzhan/lazysvn/internal/ui"
)

func main() {
	cwd := flag.String("cwd", ".", "working copy directory")
	logLimit := flag.Int("log-limit", 50, "number of log entries to show")
	flag.Parse()

	client := svn.New(*cwd)
	app := ui.NewApp(client, *logLimit)

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazysvn: %v\n", err)
		os.Exit(1)
	}
}
