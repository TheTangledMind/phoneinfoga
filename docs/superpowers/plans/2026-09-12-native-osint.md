# Native phone OSINT implementation plan

Approved design: native providers, optional terminal menu, editable JSON Google query lists, private Markdown export, and compatible Google companion. No subprocess-based provider integration. No commits, pushes, lab changes, real-number tests or credentials in the development checkout.

1. Add a bounded, validated search-list parser with number placeholders; preserve legacy defaults when no file is supplied. Test invalid files, disabled entries, expansion and driver parity.
2. Add native SerpAPI, GitHub, Reddit OAuth and DuckDuckGo instant-answer scanners using the existing scanner interface. Explicit statuses, bounded HTTP requests, no retry on blocks, safe text/URLs, fixture tests. Hudson Rock phone lookup is unsupported rather than a username lookup presented as a phone result.
3. Add source selection and Markdown export to scan; build an optional menu on the same scan path. External env files, no settings containing secrets, no overwrite of reports. Test offline menu, selection and report escaping/permissions.
4. Add the identical search-list format to the driver without changing its pinned default generator. Correct SearchPhone's misleading provider statuses and document limitations.
5. Build and run focused offline tests, existing regression tests where feasible, and a light diff/security check. Update usage documentation and give per-repository commit summaries.
