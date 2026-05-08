package svn

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("svn"); err != nil {
		t.Skip("svn not on PATH")
	}
	if _, err := exec.LookPath("svnadmin"); err != nil {
		t.Skip("svnadmin not on PATH")
	}
	dir := t.TempDir()
	cmd := exec.Command("bash", "../../testdata/repo-setup.sh", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("repo-setup.sh failed: %v\n%s", err, out)
	}
	return filepath.Join(dir, "wc")
}

func TestClientStatus(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	entries, err := c.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	byPath := map[string]Status{}
	for _, e := range entries {
		byPath[e.Path] = e.Status
	}
	checks := map[string]Status{
		"src/modified.sv":  Modified,
		"src/added.sv":     Added,
		"src/deleted.sv":   Deleted,
		"src/untracked.sv": Untracked,
	}
	for path, want := range checks {
		got, ok := byPath[path]
		if !ok {
			t.Errorf("missing entry for %s", path)
			continue
		}
		if got != want {
			t.Errorf("%s: got status %d, want %d", path, got, want)
		}
	}
}

func TestClientLog(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	entries, err := c.Log(10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Revision != 2 || entries[1].Revision != 1 {
		t.Errorf("revisions: got %d,%d, want 2,1",
			entries[0].Revision, entries[1].Revision)
	}
}

func TestClientDiff(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	d, err := c.Diff("src/modified.sv")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(d, "modified body") {
		t.Errorf("diff missing new content; got:\n%s", d)
	}
}

func TestClientDiffRevision(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	d, err := c.DiffRevision(2)
	if err != nil {
		t.Fatalf("DiffRevision: %v", err)
	}
	if !strings.Contains(d, "second rev") {
		t.Errorf("diff for r2 missing expected content; got:\n%s", d)
	}
}

func TestClientRevert(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	if err := c.Revert([]string{"src/modified.sv"}); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(wc, "src/modified.sv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "original modified") {
		t.Errorf("file not reverted; body:\n%s", body)
	}
}

func TestClientAdd(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	if err := c.Add([]string{"src/untracked.sv"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entries, err := c.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Path == "src/untracked.sv" && e.Status == Added {
			return
		}
	}
	t.Errorf("src/untracked.sv not in Added state after Add()")
}

func TestClientCommit(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	if err := c.Commit([]string{"src/modified.sv"}, "test commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	entries, err := c.Log(5)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Message != "test commit" {
		t.Errorf("top log message: got %q, want %q",
			entries[0].Message, "test commit")
	}
}

func TestClientUpdate(t *testing.T) {
	wc := setupRepo(t)
	c := New(wc)
	s, err := c.Update()
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.Revision != 2 {
		t.Errorf("Update revision: got %d, want 2", s.Revision)
	}
}

func TestClientLogEmptyRepo(t *testing.T) {
	if _, err := exec.LookPath("svn"); err != nil {
		t.Skip("svn not on PATH")
	}
	if _, err := exec.LookPath("svnadmin"); err != nil {
		t.Skip("svnadmin not on PATH")
	}
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wc := filepath.Join(dir, "wc")
	if out, err := exec.Command("svnadmin", "create", repo).CombinedOutput(); err != nil {
		t.Fatalf("svnadmin create: %v\n%s", err, out)
	}
	if out, err := exec.Command("svn", "checkout", "file://"+repo, wc).CombinedOutput(); err != nil {
		t.Fatalf("svn checkout: %v\n%s", err, out)
	}
	c := New(wc)
	entries, err := c.Log(10)
	if err != nil {
		t.Fatalf("Log on empty repo: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Log on empty repo: got %d entries, want 0", len(entries))
	}
}
