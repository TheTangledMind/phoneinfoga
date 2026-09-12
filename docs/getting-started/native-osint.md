# Native searches, terminal menu and Markdown reports

This fork implements providers in Go; it does not launch SearchPhone or other OSINT tools.
The additional features are available through the CLI/menu. The existing web UI has not been redesigned.

## Build and launch

Use a supported Go toolchain (tested with Go 1.27.1). For the terminal-only build:

```bash
go build -tags headless -trimpath -o bin/phoneinfoga .
./bin/phoneinfoga menu
```

The `headless` tag omits `serve` and the Vue web assets. It avoids needing a JavaScript build for terminal use. The original full/web build instructions still apply without this tag. This creates a development binary only; it does not replace an installed lab executable.

The menu starts with local metadata and offline Google query links. Enter a number, select sources, run, view, or save the last results. Settings accepts a query-list path and a private env-file path. Menu choices last for this session; credentials are never saved by the menu. Selected online services receive the number only after confirming the run. Ctrl+C exits scan/menu; no browser or external tool is launched.

## Sources and requirements

| Source | Behaviour | Configuration |
|---|---|---|
| `local` | Number metadata; no network | None |
| `googlesearch` | Generate Google query links; no network | Optional search list |
| `googlecse` | Existing Google Custom Search scanner | `GOOGLE_API_KEY`, `GOOGLECSE_CX` |
| `numverify` | Existing metadata API | `NUMVERIFY_API_KEY` (`NUMVERIFY_KEY` accepted by scan/menu) |
| `ovh` | Existing OVH numbering metadata | Existing scanner configuration |
| `serpapi` | Google organic results | `SERPAPI_KEY` |
| `github` | Public GitHub code matches only | `GITHUB_TOKEN` |
| `reddit` | Reddit post search using OAuth | `REDDIT_ACCESS_TOKEN`, `REDDIT_USER_AGENT` |
| `duckduckgo` | Instant answers, not full web results | None |

The new network providers are opt-in through `--sources` or the menu. Legacy `scan` defaults remain; new unselected providers are shown as skipped. Missing credentials are skipped, not counted as successful searches. `--disable` still excludes scanners. Authentication, blocking, rate limiting, invalid responses and empty results have separate statuses in the new providers. New providers fetch at most ten results per query, follow no result websites, and do not retry blocked requests. Partial provider failures produce a nonzero scan exit status after writing any requested report.

GitHub code search requires authentication ([GitHub documentation](https://docs.github.com/en/rest/search/search#search-code)). Reddit requires registered OAuth access and a descriptive User-Agent ([Reddit API guidance](https://support.reddithelp.com/hc/en-us/articles/16160319875092-Reddit-Data-API-Wiki)); supply a current token, as automatic token refresh is not implemented. SerpAPI requires its own key and account quota ([provider documentation](https://serpapi.com/search-api)). Empty DuckDuckGo instant answers do not establish that there are no web matches.

Hudson Rock phone lookup is deliberately unsupported: SearchPhone's old code passed phone numbers to a username endpoint. No claim about phone compromise can be made from that operation. Truecaller integration is not included.

## Private test credentials

Keep a private `testing.env` outside development repositories, for example under `~/.config/phoneinfoga/`. Copy the blank `.env.example` once to a new private file, give it permissions `0600`, and edit it locally. Use different test and operational credentials where possible. Do not overwrite an existing private configuration file.

```bash
./bin/phoneinfoga scan -n '+12025550123' \
  --sources local,github \
  --env-file /path/outside/repos/testing.env
```

The number above is synthetic. For live use, replace it deliberately. Searches disclose the number to the selected services. Only recognized configuration keys are loaded; file values override environment values for that scan, and later files override earlier files. No credentials are exported into the process environment. Omitting `--env-file` checks `.env` in the current working directory. Use the same external path in menu Settings. Credential values should never be pasted into command arguments, commits or chat.

## Editable Google query lists

Copy `search-list.example.json` to a private working location and edit the data:

```json
{"version":1,"queries":[
  {"category":"general","query":"\"{number}\"","enabled":true},
  {"category":"reputation","query":"\"{national}\" spam","enabled":false}
]}
```

`{number}` expands to E.164, including `+`; `{national}` is the national digits. Valid categories are `general`, `individuals`, `reputation`, `social_media`, and `disposable_providers`. Explicit `enabled: true` activates an entry. Unknown fields/categories/placeholders, control characters and oversized lists are rejected. Lists contain data only, never executable code. An explicit list replaces defaults for that run; no file keeps the original generators. Disabled entries send no searches. Limit: 100 entries and 64 KiB.

```bash
./bin/phoneinfoga scan -n '+12025550123' \
  --sources local,googlesearch --search-list search-list.example.json

./bin/phoneinfoga scan -n '+12025550123' \
  --sources serpapi --env-file /path/outside/repos/testing.env \
  --search-list search-list.example.json --max-queries 5 --pace 2s
```

Lists apply to `googlesearch`, SerpAPI, Google CSE, and the companion's `--search-list`. GitHub, Reddit and DuckDuckGo use an exact quoted E.164 query, not Google-specific syntax. For API list searches, `--max-queries` defaults to 5 (maximum 45), and `--pace` defaults to one second (minimum one second). Both API providers fetch only the first page per custom query. The browser companion has its own existing pacing/query limits. SerpAPI results retain query/category attribution; URL duplicates within a query are removed, while matches across queries/sources remain separate evidence.

The companion stays pinned to PhoneInfoga 2.11.0 for default queries; it does not depend on this checkout at runtime. Until a shared module is published, keep the small standard-library `lib/searchlist` parser and companion `internal/searchlist` parser identical. Their version-1 format and fixture tests are identical.

## Save Markdown

```bash
./bin/phoneinfoga scan -n '+12025550123' \
  --sources local,googlesearch --markdown /tmp/synthetic-phoneinfoga.md
```

Or choose **Save last results as Markdown** in the menu. Reports include the number, UTC timestamp, sources, results, and failures. Untrusted text is escaped. Files are owner-only (`0600`) and never overwrite an existing path. Prefer a location outside Git repositories; reports contain personal information. Nothing is exported automatically without the flag/menu action. Search matches are leads, not verified ownership or precise/live location.

## Validation and rollback

```bash
go test -race -tags headless ./cmd ./lib/...
go vet -tags headless ./cmd ./lib/...
```

Tests use synthetic data, saved JSON fixtures and mocked HTTP transports. Menu, report permissions/escaping, external credential isolation, provider statuses, list expansion/caps and legacy query tests are covered. Live providers, real credentials and web UI behaviour have not been validated in this change. No dependency was added.

Run the old installed lab executable for rollback: development builds do not replace it. After pushing your reviewed commits, update lab source with `git pull --ff-only`, build into a new binary path, and test before deliberately switching the operational executable. Keep the preceding executable for rollback; pulling source alone does not update a Go executable.

Google CSE pagination starts at index 1, following [Google’s API reference](https://developers.google.com/custom-search/v1/reference/rest/v1/cse/list). Requests have a timeout and stop on empty/incomplete pages instead of continuing indefinitely.
