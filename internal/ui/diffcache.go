package ui

import (
	"fmt"
	"sync"
)

type diffCache struct {
	mu    sync.Mutex
	files map[string]string // key: working-copy path; dropped on refresh
	revs  map[string]string // key: "<rev>|<path>" ("" path = repo-wide); retained
}

func newDiffCache() *diffCache {
	return &diffCache{
		files: map[string]string{},
		revs:  map[string]string{},
	}
}

func (c *diffCache) getFile(path string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.files[path]
	return v, ok
}

func (c *diffCache) setFile(path, diff string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[path] = diff
}

func revKey(rev int64, path string) string {
	return fmt.Sprintf("%d|%s", rev, path)
}

func (c *diffCache) getRev(rev int64, path string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.revs[revKey(rev, path)]
	return v, ok
}

func (c *diffCache) setRev(rev int64, path, diff string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revs[revKey(rev, path)] = diff
}

// clearFiles drops cached file diffs. Working-copy state can change between
// refreshes, so per-path cache is invalidated on every refresh. Revision
// diffs are immutable and retained.
func (c *diffCache) clearFiles() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = map[string]string{}
}
