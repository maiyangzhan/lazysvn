package ui

import "sync"

type diffCache struct {
	mu    sync.Mutex
	files map[string]string
	revs  map[int64]string
}

func newDiffCache() *diffCache {
	return &diffCache{
		files: map[string]string{},
		revs:  map[int64]string{},
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

func (c *diffCache) getRev(rev int64) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.revs[rev]
	return v, ok
}

func (c *diffCache) setRev(rev int64, diff string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revs[rev] = diff
}

// clearFiles drops cached file diffs. Working-copy state can change between
// refreshes, so per-path cache is invalidated on every refresh. Revision
// diffs are immutable and retained.
func (c *diffCache) clearFiles() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = map[string]string{}
}
