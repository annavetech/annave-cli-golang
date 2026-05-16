<!--
title:       ANNÁVE CLI — Contributing Guide
description: How to add a module or rule, run tests, submit a pull request,
             and what code style is expected.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Contributing

## Setup

```bash
git clone https://github.com/annavetech/annave-cli-golang
cd annave-cli-golang

# Build
go build ./cmd/annave

# Run tests
go test ./...

# Run a command locally
go run ./cmd/annave log analyze --help
```

Go 1.24 or later is required.

---

## Running tests

```bash
# All tests
go test ./...

# Single package with verbose output
go test -v ./internal/log/...

# Single test function
go test -v -run TestAnalyzer_Analyze ./internal/log/...

# Race detector
go test -race ./...
```

---

## Code style

- SPDX header on every `.go` file:
  ```go
  // Copyright 2026 Anna Veretennykova
  //
  // SPDX-License-Identifier: Apache-2.0
  ```
- Comments only when the reason is non-obvious — not what the code does, but why it does it that way.
- No test file uses mocks for the filesystem, Kubernetes API, or AWS SDK. Tests call the domain logic directly or use real fixtures. For adapter tests that require external APIs (AWS Cost Explorer, live k8s cluster), use build tags or skip when the dependency is unavailable.
- `gofmt` is the formatter. No additional linters are required, but `go vet ./...` must pass.
- Error codes must be added to `internal/shared/config/messages.yaml` before they are used in Go code. Do not hardcode message strings in `.go` files — use the helpers in `internal/shared/errors`.
- Every command must support `--format plain|json|table`. The plain format is always the default.

---

## How to add a new module

See `docs/ARCHITECTURE.md` for the full walkthrough. Short version:

1. Create `internal/<module>/domain/domain.go` with the result types. No external imports.
2. Create `internal/<module>/port/port.go` with a single interface.
3. Create the adapter under `internal/<module>/<adapter>/` implementing the port interface.
4. Create `cmd/annave/cmd/<module>.go` (parent command) and `cmd/annave/cmd/<module>_<subcommand>.go` (subcommand).
5. Add any configurable limits to `internal/shared/config/limits.yaml`.
6. Add any new error codes to `internal/shared/config/messages.yaml`.
7. Write a test that exercises the adapter with a real fixture (not a mock).

---

## How to add a security or infra rule

See the rules section in `docs/ARCHITECTURE.md`. All rule IDs are sequential within their prefix (`SECRET001`–`SECRET012`, `GO001`–`GO010`, `TS001`–`TS005`, `K8S001`–`K8S008`, `TF001`–`TF005`, `HELM001`–`HELM005`, `K8S101`–`K8S106`). Use the next available ID.

Write a test case that triggers the new rule with a minimal fixture. A false-positive test (a snippet that looks similar but should not match) is also useful.

---

## Submitting a pull request

1. Fork and create a branch from `main`.
2. Make your changes. `go build ./...` and `go test ./...` must pass.
3. Add a test for any new behaviour.
4. Open a PR against `main`. Describe what the change does and why.
5. Do not bump the version in `cmd/annave/cmd/root.go` — that is done at release time.

---

## Versioning

The CLI follows [Semantic Versioning](https://semver.org). The version is set in `cmd/annave/cmd/root.go`:

```go
const Version = "0.1.0"
```

- Patch (0.1.x): bug fixes, new rules within an existing scan type
- Minor (0.x.0): new subcommands, new scan types, new providers
- Major (x.0.0): breaking changes to the output schema, flag names, or error code structure

---

## Reporting bugs

Open an issue at the project repository. Include:
- The command you ran (with flags, but redact any secrets)
- The full error output
- The operating system and Go version (`go version`)
- A minimal reproduction if possible
