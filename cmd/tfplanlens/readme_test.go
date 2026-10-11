package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The README is the first thing a visitor reads: keep it short and put detail
// in docs/. Fail when it grows, and when any relative link in it or in docs/
// is broken.
func TestReadmeStaysShort(t *testing.T) {
	if n := strings.Count(read(t, repoRoot+"/README.md"), "\n"); n > 70 {
		t.Errorf("README.md is %d lines (limit 70): move detail to docs/USAGE.md or another docs page", n)
	}
}

func TestReadmeAndDocsLinksResolve(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)\s#]+)(?:#[^)]*)?\)`)
	files, _ := filepath.Glob(repoRoot + "/docs/*.md")
	files = append(files, repoRoot+"/README.md")
	for _, f := range files {
		for _, m := range link.FindAllStringSubmatch(read(t, f), -1) {
			if strings.Contains(m[1], "://") {
				continue
			}
			if !fileExists(filepath.Join(filepath.Dir(f), m[1])) {
				t.Errorf("%s: broken link %s", f, m[1])
			}
		}
	}
}
