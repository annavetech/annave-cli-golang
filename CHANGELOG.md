# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- CI runs vet, build, tests, golangci-lint, govulncheck, go mod tidy and gofmt checks.
- Repo automation for labels, size labels, assignment and merge checks.
- Release builds with GoReleaser and a Homebrew cask.

### Fixed

- Scans no longer report a clean result when a check could not run.
- Scan output no longer leaks secret values.
- Errors print on one line, with JSON errors in JSON mode.
- Exit code 2 when a scan reports findings.
- Input checks and cancellation support.

### Changed

- The CLI is renamed to tend. It was previously published as annave-cli-golang.
- Output text is plain and short.
- The license is MIT.
