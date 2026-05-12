package svn

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/maiyangzhan/lazysvn/internal/logfile"
)

type Client struct {
	cwd string
}

func New(cwd string) *Client {
	return &Client{cwd: cwd}
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	full := append([]string{"--non-interactive"}, args...)
	cmd := exec.CommandContext(ctx, "svn", full...)
	cmd.Dir = c.cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	argstr := strings.Join(args, " ")
	logfile.Append(fmt.Sprintf("svn %s [cwd=%s] ...", argstr, c.cwd))
	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start).Round(time.Millisecond)
	if ctx.Err() == context.Canceled {
		logfile.Append(fmt.Sprintf("svn %s cancelled after %s", argstr, dur))
		return nil, fmt.Errorf("svn %s cancelled", argstr)
	}
	if err != nil {
		tail := strings.TrimSpace(stderr.String())
		if len(tail) > 400 {
			tail = tail[:400] + "..."
		}
		logfile.Append(fmt.Sprintf("svn %s FAILED in %s: %v — stderr=%q", argstr, dur, err, tail))
		return nil, fmt.Errorf("svn %v: %w: %s", args, err, stderr.String())
	}
	logfile.Append(fmt.Sprintf("svn %s ok in %s (%d bytes stdout)", argstr, dur, stdout.Len()))
	return stdout.Bytes(), nil
}
