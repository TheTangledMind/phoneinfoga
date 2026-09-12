package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/output"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/lib/searchlist"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type ScanCmdOptions struct {
	MaxQueries       int
	Pace             time.Duration
	Number           string
	DisabledScanners []string
	PluginPaths      []string
	EnvFiles         []string
	Sources          []string
	SearchList       string
	Markdown         string
}

func init() { rootCmd.AddCommand(NewScanCmd(&ScanCmdOptions{})) }
func NewScanCmd(opts *ScanCmdOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "scan", Short: "Scan a phone number", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if opts.MaxQueries < 1 || opts.MaxQueries > 45 || opts.Pace < time.Second {
			return errors.New("max-queries must be 1–45 and pace at least 1s")
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		result, errs, e := collectScan(ctx, opts)
		if e != nil {
			return e
		}
		fmt.Fprintln(color.Output, "Search matches are leads, not verified ownership.")
		if e = output.NewConsoleOutput(color.Output).Write(result, errs); e != nil {
			return e
		}
		if opts.Markdown != "" {
			if e = output.SaveMarkdown(opts.Markdown, opts.Number, result, errs); e != nil {
				return e
			}
			fmt.Fprintln(color.Output, "Private Markdown report saved.")
		}
		if len(errs) > 0 {
			return errors.New("one or more scanners failed; results may be partial")
		}
		return nil
	}}
	f := cmd.Flags()
	f.StringVarP(&opts.Number, "number", "n", "", "Phone number in international format")
	f.StringArrayVarP(&opts.DisabledScanners, "disable", "D", nil, "Scanner to skip")
	f.StringArrayVar(&opts.PluginPaths, "plugin", nil, "Extra scanner plugin")
	f.StringSliceVar(&opts.EnvFiles, "env-file", nil, "Private environment files outside the repository; defaults to .env")
	f.StringSliceVar(&opts.Sources, "sources", nil, "Only these scanners; explicitly enables selected native providers")
	f.StringVar(&opts.SearchList, "search-list", "", "JSON Google query list; replaces built-in queries for this run")
	f.StringVar(&opts.Markdown, "markdown", "", "Save a new private .md report; never overwrite")
	f.IntVar(&opts.MaxQueries, "max-queries", 5, "Maximum Google list queries per API provider (1-45)")
	f.DurationVar(&opts.Pace, "pace", time.Second, "Minimum pause between Google list API queries (at least 1s)")
	return cmd
}
func collectScan(ctx context.Context, opts *ScanCmdOptions) (map[string]interface{}, map[string]error, error) {
	if opts.MaxQueries == 0 {
		opts.MaxQueries = 5
	}
	if opts.Pace == 0 {
		opts.Pace = time.Second
	}
	if opts.MaxQueries < 1 || opts.MaxQueries > 45 || opts.Pace < time.Second {
		return nil, nil, errors.New("max-queries must be 1–45 and pace at least 1s")
	}
	if !number.IsValid(opts.Number) {
		return nil, nil, errors.New("provide a valid international phone number")
	}
	num, e := number.NewNumber(opts.Number)
	if e != nil {
		return nil, nil, errors.New("invalid number")
	}
	if opts.SearchList != "" {
		if _, e := searchlist.Load(opts.SearchList, num.E164, num.RawLocal); e != nil {
			return nil, nil, e
		}
	}
	// Read only recognized configuration into this scan; never mutate os.Environ.
	options := remote.ScannerOptions{"context": ctx, "search_list": opts.SearchList, "PHONEINFOGA_NATIVE": "false", "max_queries": opts.MaxQueries, "pace": opts.Pace}
	keys := []string{"SERPAPI_KEY", "GITHUB_TOKEN", "REDDIT_ACCESS_TOKEN", "REDDIT_USER_AGENT", "NUMVERIFY_API_KEY", "NUMVERIFY_KEY", "GOOGLE_API_KEY", "GOOGLECSE_CX", "GOOGLECSE_MAX_RESULTS"}
	for _, key := range keys {
		options[key] = os.Getenv(key)
	}
	envFiles := opts.EnvFiles
	if len(envFiles) == 0 {
		envFiles = []string{".env"}
	}
	for _, file := range envFiles {
		values, e := godotenv.Read(file)
		if e != nil {
			if len(opts.EnvFiles) == 0 && os.IsNotExist(e) {
				continue
			}
			return nil, nil, errors.New("cannot load environment file")
		}
		for _, key := range keys {
			if value, ok := values[key]; ok {
				options[key] = value
			}
		}
	}
	if options.GetStringEnv("NUMVERIFY_API_KEY") == "" {
		options["NUMVERIFY_API_KEY"] = options.GetStringEnv("NUMVERIFY_KEY")
	}
	for _, p := range opts.PluginPaths {
		if e := remote.OpenPlugin(p); e != nil {
			return nil, nil, e
		}
	}
	catalog := remote.NewLibrary(filter.NewEngine())
	remote.InitScanners(catalog)
	selected := map[string]bool{}
	for _, name := range opts.Sources {
		if catalog.GetScanner(name) == nil {
			return nil, nil, fmt.Errorf("unknown scanner %q", name)
		}
		selected[name] = true
	}
	f := filter.NewEngine()
	f.AddRule(opts.DisabledScanners...)
	if len(selected) > 0 {
		for _, s := range catalog.GetAllScanners() {
			if !selected[s.Name()] {
				f.AddRule(s.Name())
			}
		}
	}
	library := remote.NewLibrary(f)
	remote.InitScanners(library)
	if len(selected) > 0 {
		options["PHONEINFOGA_NATIVE"] = "true"
	}
	type completedScan struct {
		results map[string]interface{}
		errs    map[string]error
	}
	done := make(chan completedScan, 1)
	go func() { r, e := library.Scan(num, options); done <- completedScan{r, e} }()
	var results map[string]interface{}
	var errs map[string]error
	select {
	case <-ctx.Done():
		return nil, nil, errors.New("scan interrupted")
	case done := <-done:
		results, errs = done.results, done.errs
	}
	for _, s := range library.GetAllScanners() {
		if e := s.DryRun(*num, options); e != nil {
			results[s.Name()] = remote.SearchResponse{Status: "skipped", Note: "Required configuration is missing or invalid"}
		}
		if r, ok := results[s.Name()].(remote.SearchResponse); ok && r.Status != "completed" && r.Status != "no_results" && r.Status != "skipped" {
			errs[s.Name()] = fmt.Errorf("%s: %s", r.Status, r.Note)
		}
	}
	// Provider libraries may include credential-bearing request URLs in errors.
	for name, err := range errs {
		msg := err.Error()
		for _, key := range []string{"SERPAPI_KEY", "GITHUB_TOKEN", "REDDIT_ACCESS_TOKEN", "NUMVERIFY_API_KEY", "GOOGLE_API_KEY"} {
			if value := options.GetStringEnv(key); value != "" {
				msg = strings.ReplaceAll(msg, value, "[redacted]")
			}
		}
		errs[name] = errors.New(msg)
	}
	return results, errs, nil
}
