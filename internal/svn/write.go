package svn

import "context"

func (c *Client) Commit(ctx context.Context, paths []string, msg string) error {
	args := append([]string{"commit", "-m", msg}, paths...)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) CommitFromFile(ctx context.Context, paths []string, msgFile string) error {
	args := append([]string{"commit", "-F", msgFile}, paths...)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) Revert(ctx context.Context, paths []string) error {
	args := append([]string{"revert"}, paths...)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) Add(ctx context.Context, paths []string) error {
	args := append([]string{"add"}, paths...)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) Remove(ctx context.Context, paths []string) error {
	args := append([]string{"rm"}, paths...)
	_, err := c.run(ctx, args...)
	return err
}

func (c *Client) Resolved(ctx context.Context, paths []string) error {
	return c.Resolve(ctx, paths, "working")
}

// Resolve runs `svn resolve --accept=<mode>` on the given paths. Valid modes
// are the same as svn's --accept flag: working, base, mine-conflict,
// theirs-conflict, mine-full, theirs-full.
func (c *Client) Resolve(ctx context.Context, paths []string, mode string) error {
	args := append([]string{"resolve", "--accept", mode}, paths...)
	_, err := c.run(ctx, args...)
	return err
}
