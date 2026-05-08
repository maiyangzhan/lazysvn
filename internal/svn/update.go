package svn

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type UpdateSummary struct {
	Revision int64
	Updated  int
	Added    int
	Deleted  int
	Raw      string
}

var revRegexp = regexp.MustCompile(`revision (\d+)`)

func parseUpdate(data []byte) (UpdateSummary, error) {
	var s UpdateSummary
	s.Raw = string(data)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		// Operation lines look like "U    src/foo.sv" — single letter
		// followed by spaces. Distinguishes from header lines like
		// "Updating '.':" and "Updated to revision 45." whose second
		// character is a letter.
		if len(line) >= 2 && line[1] == ' ' {
			switch line[0] {
			case 'U':
				s.Updated++
			case 'A':
				s.Added++
			case 'D':
				s.Deleted++
			}
		}
		if strings.Contains(line, "revision ") {
			m := revRegexp.FindStringSubmatch(line)
			if len(m) == 2 {
				rev, err := strconv.ParseInt(m[1], 10, 64)
				if err != nil {
					return s, fmt.Errorf("parse revision: %w", err)
				}
				s.Revision = rev
			}
		}
	}
	return s, scanner.Err()
}

func (c *Client) Update() (UpdateSummary, error) {
	out, err := c.run("update")
	if err != nil {
		return UpdateSummary{}, err
	}
	return parseUpdate(out)
}
