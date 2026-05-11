package svn

import "testing"

func TestParseStatus(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<status>
<target path=".">
<entry path="src/foo.go">
<wc-status item="modified" props="none" revision="42">
<commit revision="42"><author>alice</author><date>2026-05-01T10:15:30.000000Z</date></commit>
</wc-status>
</entry>
<entry path="src/new.go">
<wc-status item="added" props="none"></wc-status>
</entry>
<entry path="src/gone.go">
<wc-status item="deleted" props="none"></wc-status>
</entry>
<entry path="scratch.txt">
<wc-status item="unversioned" props="none"></wc-status>
</entry>
<entry path="src/mine.go">
<wc-status item="conflicted" props="none"></wc-status>
</entry>
<entry path="src/weird.go">
<wc-status item="external" props="none"></wc-status>
</entry>
</target>
</status>`)

	entries, err := parseStatus(xml)
	if err != nil {
		t.Fatalf("parseStatus: %v", err)
	}
	want := []FileEntry{
		{Path: "src/foo.go", Status: Modified},
		{Path: "src/new.go", Status: Added},
		{Path: "src/gone.go", Status: Deleted},
		{Path: "scratch.txt", Status: Untracked},
		{Path: "src/mine.go", Status: Conflicted},
		{Path: "src/weird.go", Status: Unknown},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i, w := range want {
		if entries[i] != w {
			t.Errorf("entry %d: got %+v, want %+v", i, entries[i], w)
		}
	}
}

func TestParseStatusEmpty(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<status><target path="."></target></status>`)
	entries, err := parseStatus(xml)
	if err != nil {
		t.Fatalf("parseStatus: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want empty, got %d entries", len(entries))
	}
}

func TestParseStatusMalformed(t *testing.T) {
	if _, err := parseStatus([]byte("not xml")); err == nil {
		t.Error("expected error on malformed xml")
	}
}
