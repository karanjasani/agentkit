// Package srcline reads individual source lines for one-line evidence context.
// It caches file contents so repeated lookups within a command stay cheap.
package srcline

import (
	"os"
	"strings"
	"sync"
)

// Cache reads and caches source files by path.
type Cache struct {
	mu    sync.Mutex
	files map[string][]string
}

// New returns an empty Cache.
func New() *Cache {
	return &Cache{files: map[string][]string{}}
}

// Line returns the trimmed content of the 1-based line in file, or "" if it
// cannot be read.
func (c *Cache) Line(file string, line int) string {
	if file == "" || line <= 0 {
		return ""
	}
	c.mu.Lock()
	lines, ok := c.files[file]
	if !ok {
		lines = readLines(file)
		c.files[file] = lines
	}
	c.mu.Unlock()

	if line-1 >= len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line-1])
}

func readLines(file string) []string {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	return strings.Split(string(b), "\n")
}
