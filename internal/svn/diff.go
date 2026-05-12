package svn

import (
	"context"
	"strconv"
)

func (c *Client) Diff(ctx context.Context, path string) (string, error) {
	out, err := c.run(ctx, "diff", path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) DiffRevision(ctx context.Context, rev int64) (string, error) {
	revStr := strconv.FormatInt(rev, 10)
	out, err := c.run(ctx, "diff", "-c", revStr)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
