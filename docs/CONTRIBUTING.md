<!--
title:       tend: Contributing Guide
description: How to add a module or rule, run tests, submit a pull request,
             and what code style is expected.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Contributing

## Setup

```bash
git clone https://github.com/annavetech/tend
cd tend

# Build
go build ./cmd/tend

# Run tests
go test ./...

# Run a command locally
go run ./cmd/tend log analyze --help
```

Go 1.26 or later is required.

---

## Running tests

```bash
# All tests
go test ./...

# Single package with verbose output
go test -v ./internal/log/...

# Single test function
go test -v -run TestAnalyzeReturnsReadError ./internal/log/...

# Race detector
go test -race ./...
```

---

## Code style

- SPDX header on every `.go` file:
  ```go
  // Copyright 2026 Anna Veretennykova
  //
  // SPDX-License-Identifier: MIT
  ```
- Comments only when the reason is non-obvious: not what the code does, but why it does it that way.
- Tests call the domain logic directly or use real fixtures from `testdata/`. Kubernetes adapter tests run against a local `httptest` API server, and helm tests replay recorded command output through a fake runner. No test calls AWS or a real cluster.
- CI (`.github/workflows/ci.yml`) runs `go vet ./...`, `go build ./...`, `go test -race ./...`, golangci-lint v2.14.0 (`.golangci.yml`), govulncheck, `go mod tidy -diff`, and `gofmt -s -l .`. All of them must pass.
- Return wrapped Go errors with context. Use `internal/shared/errors` only for an error that needs a machine-readable code, such as `ERR_NOT_IMPLEMENTED`.
- Every command must support `--format plain|json|table`. The plain format is always the default.

---

## How to add a new module

See `docs/ARCHITECTURE.md` for the full walkthrough. Short version:

1. Create `internal/<module>/domain/domain.go` with the result types. No external imports.
2. Create `internal/<module>/port/port.go` with a single interface.
3. Create the adapter under `internal/<module>/<adapter>/` implementing the port interface.
4. Create `cmd/tend/cmd/<module>.go` (parent command) and `cmd/tend/cmd/<module>_<subcommand>.go` (subcommand).
5. Add any configurable limits to `internal/shared/config/limits.yaml` and a matching field in `internal/shared/config/config.go`.
6. Write a test that exercises the adapter with a real fixture.

---

## How to add a security or infra rule

See the rules section in `docs/ARCHITECTURE.md`. All rule IDs are sequential within their prefix (`SECRET001` to `SECRET012`, `GO001` to `GO010`, `TS001` to `TS005`, `K8S001` to `K8S008`, `TF001` to `TF005`, `HELM001` to `HELM005`, `K8S101` to `K8S106`). Use the next available ID.

Write a test case that triggers the new rule with a minimal fixture. A false-positive test (a snippet that looks similar but should not match) is also useful.

---

## Submitting a pull request

1. Fork and create a branch from `main`.
2. Make your changes. `go build ./...` and `go test ./...` must pass.
3. Add a test for any new behaviour.
4. Open a PR against `main`. Describe what the change does and why.
5. Do not set a version in the code. Release builds take it from the tag.

---

## Versioning

The CLI follows [Semantic Versioning](https://semver.org). The version comes from the release tag. Release builds set `var version` in `cmd/tend/main.go` with `-ldflags "-X main.version=<version>"`. Other builds read the version Go stamps into the binary: a `go install` build gets the module version, and a plain `go build` in a git checkout gets a commit-based version such as `0.0.0-20261014153852-cd0f44c91eff` (with a `+dirty` suffix when the tree has uncommitted changes). Only a build without module or VCS information prints `dev`.

- Patch (0.1.x): bug fixes, new rules within an existing scan type
- Minor (0.x.0): new subcommands, new scan types, new providers
- Major (x.0.0): breaking changes to the output schema, flag names, or error code structure

---

## Releasing

Pushing a `v*` tag to `annavetech/tend` runs `.github/workflows/release.yml`:

1. GoReleaser (`.goreleaser.yaml`) builds `tend` for darwin and linux on amd64 and arm64, publishes the GitHub release with `tar.gz` archives and `checksums.txt`, and pushes the `tend` cask to `annavetech/homebrew-tap`.
2. The job fails if the release has no assets.
3. A second job runs `go install github.com/annavetech/tend/cmd/tend@<tag>` and checks that `tend version` prints the tag.

The workflow needs the `HOMEBREW_TAP_TOKEN` repository secret: a fine-grained token with Contents read and write access to `annavetech/homebrew-tap` only.

Check the configuration and build a local snapshot before tagging:

```bash
goreleaser check
goreleaser release --snapshot --clean
rm -rf dist
```

Tag and push from an up-to-date `main`:

```bash
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

Install the result with any of:

```bash
brew install annavetech/tap/tend
gh release download v0.2.0 --repo annavetech/tend --pattern 'tend_*_darwin_arm64.tar.gz' --pattern checksums.txt
go install github.com/annavetech/tend/cmd/tend@latest
```

---

## Reporting bugs

Open an issue at the project repository. Include:
- The command you ran (with flags, but redact any secrets)
- The full error output
- The operating system and Go version (`go version`)
- A minimal reproduction if possible
