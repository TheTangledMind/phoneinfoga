package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

func markdownText(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)
	s = html.EscapeString(s)
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "#", "\\#", "!", "\\!", "|", "\\|").Replace(s)
}

// SaveMarkdown creates a private report and never overwrites an existing file.
func SaveMarkdown(path, number string, results map[string]interface{}, errs map[string]error) error {
	if strings.ToLower(filepath.Ext(path)) != ".md" {
		return errors.New("report filename must end in .md")
	}
	b, e := json.Marshal(results)
	if e != nil {
		return errors.New("cannot encode report")
	}
	var plain map[string]interface{}
	if json.Unmarshal(b, &plain) != nil {
		return errors.New("cannot encode report")
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# PhoneInfoga report\n\nNumber: %s\n\nUTC: %s\n\nSearch matches are leads, not verified ownership. Empty results do not prove absence.\n\n", markdownText(number), time.Now().UTC().Format(time.RFC3339))
	keys := make([]string, 0, len(plain))
	for k := range plain {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&out, "## %s\n\n", markdownText(k))
		writeMarkdownValue(&out, plain[k], "")
		out.WriteString("\n")
	}
	for _, k := range getSortedErrorKeys(errs) {
		fmt.Fprintf(&out, "## %s — error\n\n%s\n\n", markdownText(k), markdownText(errs[k].Error()))
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return errors.New("cannot create report: choose a new writable .md path")
	}
	_, e = io.WriteString(f, out.String())
	closeErr := f.Close()
	if e != nil || closeErr != nil {
		os.Remove(path)
		return errors.New("report write failed")
	}
	return nil
}
func writeMarkdownValue(w io.Writer, v interface{}, indent string) {
	switch v := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s- **%s:**", indent, markdownText(k))
			switch val := v[k].(type) {
			case map[string]interface{}, []interface{}:
				fmt.Fprintln(w)
				writeMarkdownValue(w, val, indent+"  ")
			default:
				fmt.Fprintf(w, " %s\n", markdownText(fmt.Sprint(val)))
			}
		}
	case []interface{}:
		if len(v) == 0 {
			fmt.Fprintf(w, "%s- None returned\n", indent)
		}
		for _, item := range v {
			fmt.Fprintf(w, "%s-\n", indent)
			writeMarkdownValue(w, item, indent+"  ")
		}
	default:
		fmt.Fprintf(w, "%s%s\n", indent, markdownText(fmt.Sprint(v)))
	}
}
