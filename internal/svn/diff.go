package svn

import "strconv"

func (c *Client) Diff(path string) (string, error) {
	out, err := c.run("diff", path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) DiffRevision(rev int64) (string, error) {
	revStr := strconv.FormatInt(rev, 10)
	out, err := c.run("diff", "-c", revStr)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
