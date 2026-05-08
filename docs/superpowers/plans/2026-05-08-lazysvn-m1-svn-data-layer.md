# lazysvn M1 — SVN Data Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `internal/svn/` — a pure-Go data layer that shells out to the `svn` CLI and returns parsed Go structs. No UI. End state: a library covered by unit + integration tests, plus a throwaway `main.go` that prints `Status()` / `Log()` output to verify against a real working copy.

**Architecture:** Two files under `internal/svn/`: `parser.go` (pure XML → struct functions, no I/O) and `client.go` (`Client` struct with one method per svn operation, shells out via `os/exec`). Parsers are unit-tested with recorded XML fixtures under `testdata/xml/`. `Client` is integration-tested against a scripted mini working copy built by `testdata/repo-setup.sh`.

**Tech Stack:** Go 1.22+, stdlib only (`encoding/xml`, `os/exec`, `testing`). No external Go dependencies in this milestone.

**Reference:** Design spec at `docs/superpowers/specs/2026-05-08-lazysvn-design.md`.

**Parent directory of this plan:** `/Users/myz/claude_work/svn_tui/lazysvn/`. All paths below are relative to this directory unless otherwise noted.

---

## File Map

| File | Purpose |
|---|---|
| `go.mod` | Module declaration, `module github.com/maiyangzhan/lazysvn`, go 1.22 |
| `Makefile` | `build`, `linux`, `test`, `clean` targets |
| `internal/svn/status.go` | `FileEntry`, `Status` enum, `(*Client).Status()`, XML parser |
| `internal/svn/log.go` | `LogEntry`, `(*Client).Log(limit)`, XML parser |
| `internal/svn/diff.go` | `(*Client).Diff(path)`, `(*Client).DiffRevision(rev)` |
| `internal/svn/write.go` | `Commit`, `Revert`, `Add`, `Remove`, `Resolved` |
| `internal/svn/update.go` | `(*Client).Update()` + summary parser |
| `internal/svn/client.go` | `Client` struct, `New(cwd string)`, shared `run` helper |
| `internal/svn/status_test.go` | Parser unit tests |
| `internal/svn/log_test.go` | Parser unit tests |
| `internal/svn/update_test.go` | Parser unit tests |
| `internal/svn/client_test.go` | Integration tests using scripted repo |
| `testdata/xml/status_basic.xml` | Recorded status output |
| `testdata/xml/log_basic.xml` | Recorded log output |
| `testdata/repo-setup.sh` | Builds a mini svnadmin repo + working copy for tests |
| `cmd/lazysvn/main.go` | Throwaway verify binary (replaced in M2) |

Splitting by responsibility (one svn concept per file) rather than one giant `client.go` — each file stays small and holds method + parser + types for its concept together. `client.go` holds only the shared plumbing.

---

## Task 1: Initialize Go module and Makefile

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.gitignore`

- [ ] **Step 1: Initialize Go module**

Run from `/Users/myz/claude_work/svn_tui/lazysvn/`:

```bash
go mod init github.com/maiyangzhan/lazysvn
```

Expected: creates `go.mod` containing `module github.com/maiyangzhan/lazysvn` and `go 1.22` (or whatever local Go version is).

- [ ] **Step 2: Create Makefile**

Create `Makefile`:

```makefile
.PHONY: build linux test clean

BINARY := lazysvn
DIST   := dist

build:
	go build -o $(BINARY) ./cmd/lazysvn

linux:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/lazysvn

test:
	go test ./...

clean:
	rm -rf $(BINARY) $(DIST)
```

- [ ] **Step 3: Create .gitignore**

Create `.gitignore`:

```
lazysvn
dist/
*.test
*.out
.DS_Store
```

- [ ] **Step 4: Verify build system works (with empty stub)**

Create `cmd/lazysvn/main.go`:

```go
package main

func main() {}
```

Run: `make build && ./lazysvn && echo OK`
Expected: builds, runs silently, prints `OK`.

Run: `make linux && file dist/lazysvn-linux-amd64`
Expected: reports `ELF 64-bit LSB executable, x86-64, ..., statically linked`.

- [ ] **Step 5: Commit**

```bash
git add go.mod Makefile .gitignore cmd/lazysvn/main.go
git commit -m "chore: bootstrap Go module, Makefile, linux cross-compile target"
```

---

## Task 2: Client struct and shared run helper

**Files:**
- Create: `internal/svn/client.go`

- [ ] **Step 1: Write the Client skeleton**

Create `internal/svn/client.go`:

```go
package svn

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Client struct {
	cwd string
}

func New(cwd string) *Client {
	return &Client{cwd: cwd}
}

func (c *Client) run(args ...string) ([]byte, error) {
	cmd := exec.Command("svn", append([]string{"--non-interactive"}, args...)...)
	cmd.Dir = c.cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("svn %v: %w: %s", args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
```

- [ ] **Step 2: Verify compile**

Run: `go build ./internal/svn/`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/svn/client.go
git commit -m "feat(svn): add Client struct and run helper"
```

---

## Task 3: Status types and parser

**Files:**
- Create: `testdata/xml/status_basic.xml`
- Create: `internal/svn/status.go`
- Create: `internal/svn/status_test.go`

- [ ] **Step 1: Write XML fixture**

Create `testdata/xml/status_basic.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<status>
<target path=".">
<entry path="src/modified.sv">
<wc-status item="modified" props="none" revision="42"></wc-status>
</entry>
<entry path="src/added.sv">
<wc-status item="added" props="none" revision="-1"></wc-status>
</entry>
<entry path="src/deleted.sv">
<wc-status item="deleted" props="none" revision="42"></wc-status>
</entry>
<entry path="src/untracked.sv">
<wc-status item="unversioned" props="none"></wc-status>
</entry>
<entry path="src/conflicted.sv">
<wc-status item="modified" props="none" revision="42"></wc-status>
</entry>
</target>
</status>
```

Note: the sample includes the five states we care about. `unversioned` in XML maps to our `Untracked`. Conflict is represented by the `<entry><wc-status>` having a `tree-conflicted="true"` attribute or a text-conflict child; for MVP we trust `item` values and will extend when we hit a real conflict case.

- [ ] **Step 2: Write the failing test**

Create `internal/svn/status_test.go`:

```go
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
```

- [ ] **Step 3: Run test to confirm it fails**

Run: `go test ./internal/svn/ -run TestParseStatus -v`
Expected: FAIL, compile error about undefined `parseStatus`, `FileEntry`, `Status`, `Modified`, etc.

- [ ] **Step 4: Write minimal implementation**

Create `internal/svn/status.go`:

```go
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
```

- [ ] **Step 5: Run tests — expect PASS**

Run: `go test ./internal/svn/ -run TestParseStatus -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/svn/status.go internal/svn/status_test.go testdata/xml/status_basic.xml
git commit -m "feat(svn): Status() with XML parser and unit test"
```

---

## Task 4: Log types and parser

**Files:**
- Create: `testdata/xml/log_basic.xml`
- Create: `internal/svn/log.go`
- Create: `internal/svn/log_test.go`

- [ ] **Step 1: Write XML fixture**

Create `testdata/xml/log_basic.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<log>
<logentry revision="42">
<author>alice</author>
<date>2026-04-01T10:15:30.000000Z</date>
<msg>Fix timing issue in top module</msg>
</logentry>
<logentry revision="41">
<author>bob</author>
<date>2026-03-28T14:00:00.000000Z</date>
<msg>Add new agent</msg>
</logentry>
<logentry revision="40">
<author>alice</author>
<date>2026-03-27T09:00:00.000000Z</date>
<msg></msg>
</logentry>
</log>
```

- [ ] **Step 2: Write the failing test**

Create `internal/svn/log_test.go`:

```go
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
```

- [ ] **Step 3: Run test to confirm it fails**

Run: `go test ./internal/svn/ -run TestParseLog -v`
Expected: FAIL, compile error for `parseLog`, `LogEntry`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/svn/log.go`:

```go
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
```

- [ ] **Step 5: Run test — expect PASS**

Run: `go test ./internal/svn/ -run TestParseLog -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/svn/log.go internal/svn/log_test.go testdata/xml/log_basic.xml
git commit -m "feat(svn): Log() with XML parser and unit test"
```

---

## Task 5: Diff methods

`svn diff` outputs plain unified-diff text (no XML option that's useful here). No parsing needed — we return raw bytes as a string.

**Files:**
- Create: `internal/svn/diff.go`

- [ ] **Step 1: Write implementation**

Create `internal/svn/diff.go`:

```go
package svn

import "strconv"

func (c *Client) Diff(path string) (string, error) {
	out, err := c.run("diff", path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) DiffRevision(rev int64) (string, error) {
	revStr := strconv.FormatInt(rev, 10)
	out, err := c.run("diff", "-c", revStr)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
```

Coverage for these lives in Task 9 (integration tests). We don't unit-test a function whose only job is string-passing through `exec`.

- [ ] **Step 2: Verify compile**

Run: `go build ./internal/svn/`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/svn/diff.go
git commit -m "feat(svn): Diff() and DiffRevision()"
```

---

## Task 6: Write operations (Commit, Revert, Add, Remove, Resolved)

These are thin wrappers: build argv, fork `svn`, propagate error. No parsing. Covered by integration tests in Task 9.

**Files:**
- Create: `internal/svn/write.go`

- [ ] **Step 1: Write implementation**

Create `internal/svn/write.go`:

```go
package svn

func (c *Client) Commit(paths []string, msg string) error {
	args := append([]string{"commit", "-m", msg}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Revert(paths []string) error {
	args := append([]string{"revert"}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Add(paths []string) error {
	args := append([]string{"add"}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Remove(paths []string) error {
	args := append([]string{"rm"}, paths...)
	_, err := c.run(args...)
	return err
}

func (c *Client) Resolved(paths []string) error {
	args := append([]string{"resolved"}, paths...)
	_, err := c.run(args...)
	return err
}
```

- [ ] **Step 2: Verify compile**

Run: `go build ./internal/svn/`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/svn/write.go
git commit -m "feat(svn): Commit/Revert/Add/Remove/Resolved"
```

---

## Task 7: Update with summary parser

`svn update` output looks like:

```
Updating '.':
U    src/foo.sv
A    src/bar.sv
D    src/old.sv
Updated to revision 45.
```

We parse the last line for the new revision and count the operation letters. Expose a `UpdateSummary` struct.

**Files:**
- Create: `internal/svn/update.go`
- Create: `internal/svn/update_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/svn/update_test.go`:

```go
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
```

- [ ] **Step 2: Run test to confirm it fails**

Run: `go test ./internal/svn/ -run TestParseUpdate -v`
Expected: FAIL, compile error for `parseUpdate`, `UpdateSummary`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/svn/update.go`:

```go
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
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/svn/ -run TestParseUpdate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/svn/update.go internal/svn/update_test.go
git commit -m "feat(svn): Update() with summary parser"
```

---

## Task 8: Scripted mini-repo setup

A shell script that creates a throwaway svnadmin repo and checks out a working copy with a few files in each status state. Used by integration tests and by the manual verify binary.

**Prerequisites:** host (macOS) has `svn` and `svnadmin` installed. Verify with `command -v svnadmin`. If missing, install via `brew install subversion`.

**Files:**
- Create: `testdata/repo-setup.sh`

- [ ] **Step 1: Write the script**

Create `testdata/repo-setup.sh`:

```bash
#!/usr/bin/env bash
# Usage: ./testdata/repo-setup.sh <target-dir>
# Creates <target-dir>/repo (svnadmin repo) and <target-dir>/wc (working copy)
# with files in modified / added / deleted / untracked states.
set -euo pipefail

TARGET="${1:?target dir required}"
mkdir -p "$TARGET"
REPO="$TARGET/repo"
WC="$TARGET/wc"

rm -rf "$REPO" "$WC"

svnadmin create "$REPO"
svn checkout "file://$REPO" "$WC" >/dev/null

cd "$WC"
mkdir -p src
echo "original modified" > src/modified.sv
echo "to stay"            > src/unchanged.sv
echo "to delete"          > src/deleted.sv
svn add src >/dev/null
svn commit -m "initial" >/dev/null

# Produce a second revision so Log() has 2 entries.
echo "second rev" > src/unchanged.sv
svn commit -m "second revision" >/dev/null

# Now create the status states the tests expect.
echo "modified body" > src/modified.sv
echo "brand new"     > src/added.sv
svn add src/added.sv >/dev/null
svn rm src/deleted.sv >/dev/null
echo "untracked body" > src/untracked.sv

echo "$WC"
```

- [ ] **Step 2: Make it executable and smoke test**

Run:

```bash
chmod +x testdata/repo-setup.sh
tmp=$(mktemp -d)
bash testdata/repo-setup.sh "$tmp"
svn status "$tmp/wc"
rm -rf "$tmp"
```

Expected output from `svn status` shows `M src/modified.sv`, `A src/added.sv`, `D src/deleted.sv`, `? src/untracked.sv`.

- [ ] **Step 3: Commit**

```bash
git add testdata/repo-setup.sh
git commit -m "test: add scripted mini-repo builder for integration tests"
```

---

## Task 9: Client integration tests

Drive each `Client` method against the scripted repo. This is the test surface for `Commit`/`Revert`/`Add`/`Remove`/`Resolved`/`Diff`/`DiffRevision` plus an end-to-end check of `Status`/`Log`/`Update`.

**Files:**
- Create: `internal/svn/client_test.go`

- [ ] **Step 1: Write the test harness**

Create `internal/svn/client_test.go`:

```go
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
```

Note: `Remove` and `Resolved` are not covered by dedicated tests here — `Remove` is exercised by the repo-setup script (uses `svn rm`) and `Resolved` requires constructing a conflict, which is more work than the MVP wants. Both are thin wrappers over `run()` with the same shape as `Add`, so confidence comes from shared code paths. A test for them is a good M1.5 follow-up if conflicts show up in real use.

- [ ] **Step 2: Run the integration tests**

Run: `go test ./internal/svn/ -v`
Expected: all tests PASS. On macOS you may see a warning from svn about config files — safe to ignore.

- [ ] **Step 3: Commit**

```bash
git add internal/svn/client_test.go
git commit -m "test(svn): integration tests against scripted working copy"
```

---

## Task 10: Throwaway verify binary and linux build check

Replace the empty `main.go` stub with one that exercises `Status()` and `Log()` against the current directory. This gives a manual smoke test and proves the `make linux` binary actually works when carried to a server. Will be overwritten in M2.

**Files:**
- Modify: `cmd/lazysvn/main.go`

- [ ] **Step 1: Replace main.go**

Overwrite `cmd/lazysvn/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maiyangzhan/lazysvn/internal/svn"
)

func main() {
	cwd := flag.String("cwd", ".", "working copy directory")
	limit := flag.Int("log-limit", 5, "number of log entries to show")
	flag.Parse()

	c := svn.New(*cwd)

	fmt.Println("=== svn status ===")
	entries, err := c.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		os.Exit(1)
	}
	for _, e := range entries {
		fmt.Printf("%d  %s\n", e.Status, e.Path)
	}

	fmt.Println("=== svn log ===")
	logs, err := c.Log(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log: %v\n", err)
		os.Exit(1)
	}
	for _, l := range logs {
		fmt.Printf("r%d  %s  %s  %s\n", l.Revision, l.Author, l.Date.Format("2006-01-02"), l.Message)
	}
}
```

- [ ] **Step 2: Verify host build + run against the scripted repo**

Run:

```bash
tmp=$(mktemp -d)
bash testdata/repo-setup.sh "$tmp"
make build
./lazysvn --cwd "$tmp/wc" --log-limit 3
rm -rf "$tmp"
```

Expected: prints 5 status entries (Status numeric value 1-4, matching the enum) and 2 log lines (r2 then r1).

- [ ] **Step 3: Verify linux build and binary is static**

Run: `make linux && file dist/lazysvn-linux-amd64`
Expected: `ELF 64-bit LSB executable, x86-64, ..., statically linked`.

- [ ] **Step 4: Run the full test suite once more**

Run: `make test`
Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/lazysvn/main.go
git commit -m "feat(cmd): verify binary prints Status() and Log()"
```

---

## Milestone verification

At this point the repo state satisfies M1's exit criteria:

- `internal/svn/` exposes the full method set the UI layer will need in M2.
- XML parsers and the `Update()` text parser each have unit tests with fixtures.
- Every `Client` method except `Remove`/`Resolved` has an integration test.
- `make linux` produces a static ELF binary that runs the verify harness against a real working copy.

**Manual final check — carry the binary to a real server:**

1. Run `make linux` on the Mac.
2. Transfer `dist/lazysvn-linux-amd64` to an actual company server (or any Linux host with svn and a working copy).
3. `./lazysvn-linux-amd64 --cwd /path/to/working-copy --log-limit 10` — confirm it prints real status and log entries.

Any unexpected parse error against real output is a fixture gap — record the actual XML into a new fixture under `testdata/xml/` and extend the parser before starting M2.
