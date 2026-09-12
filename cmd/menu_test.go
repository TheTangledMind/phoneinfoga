package cmd

import (
	"bytes"
	"context"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenuOfflineScanAndEOF(t *testing.T) {
	var out bytes.Buffer
	if e := runMenu(context.Background(), strings.NewReader("1\n+12025550123\n4\n0\n"), &out); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(out.String(), "local") || !strings.Contains(out.String(), "googlesearch") {
		t.Fatal(out.String())
	}
	if e := runMenu(context.Background(), strings.NewReader(""), &out); e != nil {
		t.Fatal(e)
	}
}
func TestSourcesAndCustomListError(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	_, _, e := collectScan(context.Background(), &ScanCmdOptions{Number: "+12025550123", Sources: []string{"unknown"}})
	if e == nil {
		t.Fatal("unknown source accepted")
	}
	r, _, e := collectScan(context.Background(), &ScanCmdOptions{Number: "+12025550123", Sources: []string{"local", "github"}})
	if e != nil {
		t.Fatal(e)
	}
	if r["github"].(remote.SearchResponse).Status != "skipped" {
		t.Fatal(r)
	}
}

func TestPrivateEnvDoesNotPersist(t *testing.T) {
	t.Setenv("SERPAPI_KEY", "")
	os.Unsetenv("SERPAPI_KEY")
	path := filepath.Join(t.TempDir(), "private.env")
	os.WriteFile(path, []byte("SERPAPI_KEY=fixture-only\n"), 0600)
	_, _, e := collectScan(context.Background(), &ScanCmdOptions{Number: "+12025550123", Sources: []string{"local"}, EnvFiles: []string{path}})
	if e != nil {
		t.Fatal(e)
	}
	if os.Getenv("SERPAPI_KEY") != "" {
		t.Fatal("private credentials leaked into process environment")
	}
}
