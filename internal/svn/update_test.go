package svn

import "testing"

func TestParseUpdate(t *testing.T) {
	in := `Updating '.':
U    src/foo.sv
A    src/bar.sv
A    src/baz.sv
D    src/old.sv
Updated to revision 45.
`
	s, err := parseUpdate([]byte(in))
	if err != nil {
		t.Fatalf("parseUpdate: %v", err)
	}
	if s.Revision != 45 {
		t.Errorf("Revision: got %d, want 45", s.Revision)
	}
	if s.Updated != 1 || s.Added != 2 || s.Deleted != 1 {
		t.Errorf("counts: got U=%d A=%d D=%d, want 1/2/1",
			s.Updated, s.Added, s.Deleted)
	}
}

func TestParseUpdateAtRevision(t *testing.T) {
	in := `Updating '.':
At revision 45.
`
	s, err := parseUpdate([]byte(in))
	if err != nil {
		t.Fatalf("parseUpdate: %v", err)
	}
	if s.Revision != 45 {
		t.Errorf("Revision: got %d, want 45", s.Revision)
	}
	if s.Updated+s.Added+s.Deleted != 0 {
		t.Errorf("expected zero counts, got %+v", s)
	}
}
