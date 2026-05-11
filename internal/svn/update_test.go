package svn

import "testing"

func TestParseUpdate(t *testing.T) {
	out := []byte(`Updating '.':
U    src/foo.go
U    src/bar.go
A    src/new.go
D    src/old.go
Updated to revision 45.
`)
	s, err := parseUpdate(out)
	if err != nil {
		t.Fatalf("parseUpdate: %v", err)
	}
	if s.Revision != 45 {
		t.Errorf("Revision = %d, want 45", s.Revision)
	}
	if s.Updated != 2 {
		t.Errorf("Updated = %d, want 2", s.Updated)
	}
	if s.Added != 1 {
		t.Errorf("Added = %d, want 1", s.Added)
	}
	if s.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", s.Deleted)
	}
	if s.Raw != string(out) {
		t.Errorf("Raw not preserved")
	}
}

func TestParseUpdateAtRevision(t *testing.T) {
	// `svn update` on a fully up-to-date WC prints "At revision N."
	out := []byte(`Updating '.':
At revision 45.
`)
	s, err := parseUpdate(out)
	if err != nil {
		t.Fatalf("parseUpdate: %v", err)
	}
	if s.Revision != 45 {
		t.Errorf("Revision = %d, want 45", s.Revision)
	}
	if s.Updated != 0 || s.Added != 0 || s.Deleted != 0 {
		t.Errorf("counts should be zero, got U=%d A=%d D=%d", s.Updated, s.Added, s.Deleted)
	}
}

func TestParseUpdateIgnoresHeaders(t *testing.T) {
	// Header lines like "Updating '.':" must not inflate counts even
	// though they start with 'U'.
	out := []byte(`Updating '.':
Updated to revision 1.
`)
	s, err := parseUpdate(out)
	if err != nil {
		t.Fatalf("parseUpdate: %v", err)
	}
	if s.Updated != 0 {
		t.Errorf("Updated = %d, want 0 (header line should not count)", s.Updated)
	}
	if s.Revision != 1 {
		t.Errorf("Revision = %d, want 1", s.Revision)
	}
}

func TestParseUpdateEmpty(t *testing.T) {
	s, err := parseUpdate([]byte(""))
	if err != nil {
		t.Fatalf("parseUpdate: %v", err)
	}
	if s.Revision != 0 || s.Updated != 0 || s.Added != 0 || s.Deleted != 0 {
		t.Errorf("want zero summary, got %+v", s)
	}
}
