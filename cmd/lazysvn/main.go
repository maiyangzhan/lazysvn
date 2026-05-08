package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

func main() {
	cwd := flag.String("cwd", ".", "working copy directory")
	limit := flag.Int("log-limit", 5, "number of log entries to show")
	flag.Parse()

	c := svn.New(*cwd)

	fmt.Println("=== svn status ===")
	entries, err := c.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		os.Exit(1)
	}
	for _, e := range entries {
		fmt.Printf("%d  %s\n", e.Status, e.Path)
	}

	fmt.Println("=== svn log ===")
	logs, err := c.Log(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log: %v\n", err)
		os.Exit(1)
	}
	for _, l := range logs {
		fmt.Printf("r%d  %s  %s  %s\n", l.Revision, l.Author, l.Date.Format("2006-01-02"), l.Message)
	}
}
