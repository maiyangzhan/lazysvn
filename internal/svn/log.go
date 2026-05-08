package svn

import (
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

func (c *Client) Log(limit int) ([]LogEntry, error) {
	out, err := c.run("log", "--xml", "--limit", strconv.Itoa(limit))
	if err != nil {
		return nil, err
	}
	return parseLog(out)
}
