package svn

import (
	"os"
	"testing"
)

func TestParseStatus(t *testing.T) {
	data, err := os.ReadFile("../../testdata/xml/status_basic.xml")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseStatus(data)
	if err != nil {
		t.Fatalf("parseStatus: %v", err)
	}
	want := []FileEntry{
		{Path: "src/modified.sv", Status: Modified},
		{Path: "src/added.sv", Status: Added},
		{Path: "src/deleted.sv", Status: Deleted},
		{Path: "src/untracked.sv", Status: Untracked},
		{Path: "src/conflicted.sv", Status: Modified},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entry %d: got %+v, want %+v", i, entries[i], want[i])
		}
	}
}
