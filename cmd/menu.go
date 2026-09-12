package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/sundowndev/phoneinfoga/v2/lib/output"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{Use: "menu", Short: "Interactive phone searches and private Markdown reports", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runMenu(ctx, os.Stdin, os.Stdout)
	}})
}
func runMenu(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewScanner(in)
	reader.Buffer(make([]byte, 1024), 65536)
	read := func(prompt string) (string, bool) {
		fmt.Fprint(out, prompt)
		done := make(chan bool, 1)
		go func() { done <- reader.Scan() }()
		select {
		case <-ctx.Done():
			return "", false
		case ok := <-done:
			if !ok {
				return "", false
			}
		}
		return strings.TrimSpace(reader.Text()), true
	}
	opts := ScanCmdOptions{Sources: []string{"local", "googlesearch"}}
	var results map[string]interface{}
	var errs map[string]error
	reportNumber := ""
	for {
		if ctx.Err() != nil {
			return nil
		}
		fmt.Fprintf(out, "\nPhoneInfoga\nNumber: %s\nSources: %s\n1. Enter number\n2. Select sources\n3. Settings (query list and private credentials file)\n4. Run search\n5. View last results\n6. Save last results as Markdown\n0. Exit\n", opts.Number, strings.Join(opts.Sources, ", "))
		choice, ok := read("Choice: ")
		if !ok {
			if ctx.Err() != nil {
				return nil
			}
			return reader.Err()
		}
		switch choice {
		case "0":
			return nil
		case "1":
			v, ok := read("International number (+country code): ")
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return reader.Err()
			}
			opts.Number = v
		case "2":
			fmt.Fprintln(out, "Available: local, googlesearch (offline query links), numverify, ovh, googlecse, serpapi, github, reddit, duckduckgo (instant answers only)")
			v, ok := read("Comma-separated sources: ")
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return reader.Err()
			}
			if v == "" {
				fmt.Fprintln(out, "Keep at least one source.")
				continue
			}
			opts.Sources = strings.Split(v, ",")
			for i := range opts.Sources {
				opts.Sources[i] = strings.TrimSpace(opts.Sources[i])
			}
		case "3":
			v, ok := read("Google query JSON path (blank = built-in defaults): ")
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return reader.Err()
			}
			opts.SearchList = v
			v, ok = read("Private env-file path (blank = default .env): ")
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return reader.Err()
			}
			opts.EnvFiles = nil
			if v != "" {
				opts.EnvFiles = []string{v}
			}
		case "4":
			external := false
			for _, s := range opts.Sources {
				if s != "local" && s != "googlesearch" {
					external = true
				}
			}
			if external {
				v, ok := read("Selected online sources will receive this number. Continue? [y/N]: ")
				if !ok {
					if ctx.Err() != nil {
						return nil
					}
					return reader.Err()
				}
				if strings.ToLower(v) != "y" {
					continue
				}
			}
			r, e, err := collectScan(ctx, &opts)
			if err != nil {
				fmt.Fprintln(out, "Cannot scan:", err)
				continue
			}
			results, errs, reportNumber = r, e, opts.Number
			fmt.Fprintln(out, "Search matches are leads, not verified ownership.")
			if err := output.NewConsoleOutput(out).Write(results, errs); err != nil {
				return err
			}
		case "5":
			if results == nil {
				fmt.Fprintln(out, "Run a search first.")
			} else {
				if e := output.NewConsoleOutput(out).Write(results, errs); e != nil {
					return e
				}
			}
		case "6":
			if results == nil {
				fmt.Fprintln(out, "Run a search first.")
				continue
			}
			v, ok := read("New .md file path (prefer outside repositories): ")
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return reader.Err()
			}
			if e := output.SaveMarkdown(v, reportNumber, results, errs); e != nil {
				fmt.Fprintln(out, e)
			} else {
				fmt.Fprintln(out, "Private Markdown report saved.")
			}
		default:
			fmt.Fprintln(out, "Choose 0–6.")
		}
	}
}
