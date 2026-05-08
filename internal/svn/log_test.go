package svn

import (
	"os"
	"testing"
	"time"
)

func TestParseLog(t *testing.T) {
	data, err := os.ReadFile("../../testdata/xml/log_basic.xml")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseLog(data)
	if err != nil {
		t.Fatalf("parseLog: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0].Revision != 42 || entries[0].Author != "alice" ||
		entries[0].Message != "Fix timing issue in top module" {
		t.Errorf("entry 0 wrong: %+v", entries[0])
	}
	wantDate, _ := time.Parse(time.RFC3339Nano, "2026-04-01T10:15:30.000000Z")
	if !entries[0].Date.Equal(wantDate) {
		t.Errorf("entry 0 date: got %v, want %v", entries[0].Date, wantDate)
	}
	if entries[2].Message != "" {
		t.Errorf("entry 2 should have empty message, got %q", entries[2].Message)
	}
}
