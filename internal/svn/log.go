package svn

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"
)

type LogEntry struct {
	Revision int64
	Author   string
	Date     time.Time
	Message  string
}

type xmlLog struct {
	Entries []struct {
		Revision string `xml:"revision,attr"`
		Author   string `xml:"author"`
		Date     string `xml:"date"`
		Msg      string `xml:"msg"`
	} `xml:"logentry"`
}

func parseLog(data []byte) ([]LogEntry, error) {
	var root xmlLog
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse log xml: %w", err)
	}
	out := make([]LogEntry, 0, len(root.Entries))
	for _, e := range root.Entries {
		rev, err := strconv.ParseInt(e.Revision, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse revision %q: %w", e.Revision, err)
		}
		date, err := time.Parse(time.RFC3339Nano, e.Date)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", e.Date, err)
		}
		out = append(out, LogEntry{
			Revision: rev,
			Author:   e.Author,
			Date:     date,
			Message:  e.Msg,
		})
	}
	return out, nil
}

func (c *Client) Log(ctx context.Context, limit int) ([]LogEntry, error) {
	// Use -r HEAD:0 so we see commits in the repository even if the
	// working copy's base revision hasn't been updated (e.g. right
	// after Commit(), which advances the repo head but not the WC
	// base for the directory). Default `svn log` uses BASE:0 and
	// would miss those commits. We use HEAD:0 rather than HEAD:1 so
	// that an empty (just-created) repository where HEAD=r0 returns
	// an empty log instead of "E160006: No such revision 1".
	out, err := c.run(ctx, "log", "--xml", "--limit", strconv.Itoa(limit), "-r", "HEAD:0")
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}

// LogBefore returns up to `limit` entries strictly older than `before`.
// Used for pagination (load-more). Returns nil when nothing precedes.
func (c *Client) LogBefore(ctx context.Context, before int64, limit int) ([]LogEntry, error) {
	if before <= 1 {
		return nil, nil
	}
	out, err := c.run(ctx, "log", "--xml", "--limit", strconv.Itoa(limit),
		"-r", strconv.FormatInt(before-1, 10)+":0")
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}

// LogPath returns up to `limit` entries that touched `path`, newest first.
func (c *Client) LogPath(ctx context.Context, path string, limit int) ([]LogEntry, error) {
	out, err := c.run(ctx, "log", "--xml", "--limit", strconv.Itoa(limit), "-r", "HEAD:0", path)
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}

// LogPathBefore is the path-filtered equivalent of LogBefore.
func (c *Client) LogPathBefore(ctx context.Context, path string, before int64, limit int) ([]LogEntry, error) {
	if before <= 1 {
		return nil, nil
	}
	out, err := c.run(ctx, "log", "--xml", "--limit", strconv.Itoa(limit),
		"-r", strconv.FormatInt(before-1, 10)+":0", path)
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}
