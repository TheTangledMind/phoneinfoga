package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownPrivateAndEscaped(t *testing.T) {
	p := filepath.Join(t.TempDir(), "report.md")
	if e := SaveMarkdown(p, "+12025550123", map[string]interface{}{"fixture": map[string]string{"title": "<img src=x> [fake](file:///etc/passwd)\x1b"}}, nil); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "<img") || strings.Contains(string(b), "\x1b") {
		t.Fatal(string(b))
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	if SaveMarkdown(p, "", nil, nil) == nil {
		t.Fatal("overwrote report")
	}
}
