package svn

func (c *Client) Commit(paths []string, msg string) error {
	args := append([]string{"commit", "-m", msg}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) CommitFromFile(paths []string, msgFile string) error {
	args := append([]string{"commit", "-F", msgFile}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Revert(paths []string) error {
	args := append([]string{"revert"}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Add(paths []string) error {
	args := append([]string{"add"}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Remove(paths []string) error {
	args := append([]string{"rm"}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Resolved(paths []string) error {
	return c.Resolve(paths, "working")
}

// Resolve runs `svn resolve --accept=<mode>` on the given paths. Valid modes
// are the same as svn's --accept flag: working, base, mine-conflict,
// theirs-conflict, mine-full, theirs-full.
func (c *Client) Resolve(paths []string, mode string) error {
	args := append([]string{"resolve", "--accept", mode}, paths...)
	_, err := c.run(args...)
	return err
}
