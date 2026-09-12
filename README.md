# PhoneInfoga — fork

We forked the previously unmaintained [PhoneInfoga project by Sundowndev and contributors](https://github.com/sundowndev/phoneinfoga)
with the intention of continuing its development, expanding its phone-number
OSINT capabilities, and improving its integrations and usability.

Credit for the original project belongs to Sundowndev and its contributors.
Original GPL-3.0 licensing is preserved: [LICENSE](LICENSE).


<p align="center">
  <img src="./docs/images/banner.png" width="500" alt="Original PhoneInfoga project logo" />
</p>

## Documentation

- [Installation and source builds](docs/getting-started/install.md)
- [Usage instructions](docs/getting-started/usage.md)
- [Native providers, terminal menu, editable searches and Markdown export](docs/getting-started/native-osint.md)
- [Scanner configuration](docs/getting-started/scanners.md)
- [Go module usage](docs/getting-started/go-module-usage.md)
- [REST API specification](web/docs/swagger.yaml)
- [Original upstream documentation](https://sundowndev.github.io/phoneinfoga/)
- [Original developer’s introduction](https://medium.com/@SundownDEV/phone-number-scanning-osint-recon-tool-6ad8f0cac27b)

The original guides remain available in full. References to Sundowndev releases,
container images and package distributions refer to upstream builds, not builds
of this fork. Building this checkout is required to include local code changes.

## About

PhoneInfoga is a tool for gathering information about international phone numbers. It allows you to first gather basic information such as country, area, carrier and line type, then use various techniques to try to find the VoIP provider or identify the owner. It works with a collection of scanners that must be configured in order for the tool to be effective. PhoneInfoga doesn't automate everything, it's just there to help investigating on phone numbers.

## Features

- Validate phone-number format and retrieve available metadata
- Gather basic information such as country, line type and carrier
- OSINT footprinting using external APIs, phone books & search engines
- Check for reputation reports, social media, disposable numbers and more
- Use the graphical user interface to run scans from the browser
- Programmatic usage with the [REST API specification](web/docs/swagger.yaml) and [Go modules](https://pkg.go.dev/github.com/sundowndev/phoneinfoga/v2)

## Native terminal additions

Build this fork's terminal version with `go build -tags headless -o bin/phoneinfoga .`,
then run `./bin/phoneinfoga menu`. Native SerpAPI, GitHub, Reddit OAuth and
DuckDuckGo instant-answer providers complement the existing scanners. The menu
selects sources and saves private Markdown reports. Google query lists can be
edited as JSON and shared with the companion. See the [guide](docs/getting-started/native-osint.md)
for provider requirements, external credential files and limitations.

## Anti-features

- Does not claim to provide relevant or verified data, it's just a tool !
- Does not allow to "track" a phone or its owner in real time
- Does not allow to get the precise phone location
- Does not allow to hack a phone

## License

This tool is licensed under the [GNU General Public License v3.0](LICENSE).

[Icon](https://www.flaticon.com/free-icon/fingerprint-search-symbol-of-secret-service-investigation_48838) made by <a href="https://www.freepik.com/" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">flaticon.com</a> is licensed by <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons BY 3.0" target="_blank">CC 3.0 BY</a>.

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
