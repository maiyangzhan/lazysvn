package svn

import (
	"encoding/xml"
	"fmt"
)

type Status int

const (
	Unknown Status = iota
	Modified
	Added
	Deleted
	Untracked
	Conflicted
)

type FileEntry struct {
	Path   string
	Status Status
}

type xmlStatus struct {
	Target struct {
		Entries []struct {
			Path     string `xml:"path,attr"`
			WCStatus struct {
				Item string `xml:"item,attr"`
			} `xml:"wc-status"`
		} `xml:"entry"`
	} `xml:"target"`
}

func parseStatus(data []byte) ([]FileEntry, error) {
	var root xmlStatus
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse status xml: %w", err)
	}
	out := make([]FileEntry, 0, len(root.Target.Entries))
	for _, e := range root.Target.Entries {
		out = append(out, FileEntry{Path: e.Path, Status: mapStatus(e.WCStatus.Item)})
	}
	return out, nil
}

func mapStatus(item string) Status {
	switch item {
	case "modified":
		return Modified
	case "added":
		return Added
	case "deleted":
		return Deleted
	case "unversioned":
		return Untracked
	case "conflicted":
		return Conflicted
	}
	return Unknown
}

func (c *Client) Status() ([]FileEntry, error) {
	out, err := c.run("status", "--xml")
	if err != nil {
		return nil, err
	}
	return parseStatus(out)
}
