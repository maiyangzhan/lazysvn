package svn

import (
	"testing"
	"time"
)

func TestParseLog(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<log>
<logentry revision="42">
<author>alice</author>
<date>2026-05-01T10:15:30.123456Z</date>
<msg>fix parsing bug</msg>
</logentry>
<logentry revision="41">
<author>bob</author>
<date>2026-04-30T09:00:00.000000Z</date>
<msg>add feature</msg>
</logentry>
</log>`)

	entries, err := parseLog(xml)
	if err != nil {
		t.Fatalf("parseLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].Revision != 42 {
		t.Errorf("entry[0].Revision = %d, want 42", entries[0].Revision)
	}
	if entries[0].Author != "alice" {
		t.Errorf("entry[0].Author = %q, want alice", entries[0].Author)
	}
	if entries[0].Message != "fix parsing bug" {
		t.Errorf("entry[0].Message = %q, want %q", entries[0].Message, "fix parsing bug")
	}
	wantDate := time.Date(2026, 5, 1, 10, 15, 30, 123456000, time.UTC)
	if !entries[0].Date.Equal(wantDate) {
		t.Errorf("entry[0].Date = %v, want %v", entries[0].Date, wantDate)
	}

	if entries[1].Revision != 41 || entries[1].Author != "bob" {
		t.Errorf("entry[1] = %+v", entries[1])
	}
}

func TestParseLogEmpty(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?><log></log>`)
	entries, err := parseLog(xml)
	if err != nil {
		t.Fatalf("parseLog: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want empty, got %d", len(entries))
	}
}

func TestParseLogBadRevision(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<log><logentry revision="abc"><author>a</author><date>2026-05-01T10:15:30.000000Z</date><msg>m</msg></logentry></log>`)
	if _, err := parseLog(xml); err == nil {
		t.Error("expected error on bad revision")
	}
}

func TestParseLogBadDate(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<log><logentry revision="1"><author>a</author><date>not-a-date</date><msg>m</msg></logentry></log>`)
	if _, err := parseLog(xml); err == nil {
		t.Error("expected error on bad date")
	}
}
