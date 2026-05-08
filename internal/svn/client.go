package svn

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Client struct {
	cwd string
}

func New(cwd string) *Client {
	return &Client{cwd: cwd}
}

func (c *Client) run(args ...string) ([]byte, error) {
	cmd := exec.Command("svn", append([]string{"--non-interactive"}, args...)...)
	cmd.Dir = c.cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("svn %v: %w: %s", args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
