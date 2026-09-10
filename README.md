# PhoneInfoga — fork

Personal development fork of [PhoneInfoga by Sundowndev and contributors](https://github.com/sundowndev/phoneinfoga).
The upstream project describes itself as unmaintained. This fork is a starting
point for local improvements; it does not claim new capabilities or maintained
upstream releases. Original GPL-3.0 licensing is preserved: [LICENSE](LICENSE).

PhoneInfoga gathers phone-number metadata and generates search queries using
configurable scanners. Search matches do not verify ownership, track a phone in
real time or establish its precise location. Use it only for authorized research.

## Usage

With an existing PhoneInfoga executable:

```bash
phoneinfoga version
phoneinfoga scanners --help
phoneinfoga scan --help
phoneinfoga scan -n '+COUNTRY_CODE_AND_NUMBER'
```

If it is not on PATH, use the full path to your existing executable instead.
External scanners may send the number to their providers and need private API
configuration. Keep `.env` and credentials out of commits.

- [Installation and source build instructions](docs/getting-started/install.md)
- [Usage](docs/getting-started/usage.md)
- [Scanners and configuration](docs/getting-started/scanners.md)
- [Go library usage](docs/getting-started/go-module-usage.md)

The optional [Google driver companion fork](https://github.com/TheTangledMind/phoneinfoga-google-driver)
targets the **2.11.0** query API without replacing this executable or starting a
web server. Its browser search requires no API key.

The existing `serve` command binds beyond loopback and has no built-in
authentication. Do not expose it to the LAN or internet for local companion use;
the companion does not require it. Upstream documentation may reference upstream
binaries and deployment defaults rather than this fork.
