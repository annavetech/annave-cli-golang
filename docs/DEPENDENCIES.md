<!--
title:       ANNÁVE CLI — Dependencies
description: Every direct dependency, exact pinned version, what each does,
             and what breaks if you change the version.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Dependencies

Go version: **1.24** (minimum). The CLI uses `log/slog` (added in 1.21) and range-over-integer syntax (added in 1.22).

All dependencies are listed in `go.mod`. Run `go mod verify` to confirm checksums match `go.sum`.

---

## Direct dependencies

### `github.com/spf13/cobra v1.10.2`

CLI framework. Provides command registration, flag parsing, help text generation, and shell completion. The CLI uses `cobra.Command` exclusively — no custom argument parsing.

| Version range | Result |
|---|---|
| < v1.8.0 | Unknown — `FParseErrWhitelist` may not exist |
| v1.10.2 | Confirmed working |
| v1.11+ | Unknown — API is stable, should work |

All commands are registered in `init()` blocks using `parent.AddCommand(child)`. Never call `Execute()` more than once.

---

### `gopkg.in/yaml.v3 v3.0.1`

YAML parsing for embedded config files (`limits.yaml`, `messages.yaml`) and for parsing Kubernetes manifests in the infra validator.

| Version range | Result |
|---|---|
| v3.0.1 | Confirmed working |
| v3.0.0 | Unknown |

Do not use `gopkg.in/yaml.v2` — it is imported transitively by k8s dependencies but is not used directly. Stick to v3 for all new code.

---

### `k8s.io/client-go v0.29.3`

Kubernetes client for `annave cleanup scan` and `annave security audit --type k8s-live`. Provides the typed client (`kubernetes.Clientset`), kubeconfig loading (`tools/clientcmd`), and list/watch APIs.

| Version range | Result |
|---|---|
| v0.28.x | Unknown — API may differ |
| v0.29.3 | Confirmed working |
| v0.30+ | Unknown — check if kubeconfig loading API is unchanged |

This is the largest transitive dependency tree in the project (30+ packages). Updating it should be done carefully and tested against a real cluster.

---

### `k8s.io/api v0.29.3` and `k8s.io/apimachinery v0.29.3`

Kubernetes API type definitions used by client-go. Must be pinned to the same minor version as client-go. Do not update one without the other.

---

### `sigs.k8s.io/yaml v1.3.0`

YAML parsing for Kubernetes manifests in `annave security audit --type k8s-local` and `annave infra validate --type k8s`. This library is a thin wrapper around `go-yaml` that unmarshals into Go maps, matching what `kubectl apply` parses.

It is a transitive dependency of client-go, so it is always present. Using it directly avoids a separate dependency for manifest parsing.

---

### `github.com/aws/aws-sdk-go-v2/config v1.32.17`

AWS SDK v2 configuration loading for `annave cost scan`. Implements the credential chain: environment variables → shared credentials file → IAM instance role.

---

### `github.com/aws/aws-sdk-go-v2/service/costexplorer v1.63.8`

AWS Cost Explorer client. Used for two API calls: `GetCostAndUsage` (daily costs by service) and `GetCostForecast` (30-day projected spend). The Cost Explorer API is a global endpoint; it does not require a region to be set in the config.

Note: Cost Explorer charges $0.01 per API request. Each `annave cost scan` call makes 1 or 2 requests (cost data + optional forecast).

---

## Indirect dependencies (selected)

These are required by direct dependencies but not imported directly.

| Package | Version | Required by |
|---|---|---|
| `github.com/aws/aws-sdk-go-v2` | v1.41.7 | all aws/* packages — core retry, signing |
| `github.com/aws/smithy-go` | v1.25.1 | aws-sdk-go-v2 — HTTP transport and signing |
| `github.com/spf13/pflag` | v1.0.9 | cobra — POSIX flag parsing |
| `golang.org/x/net` | v0.19.0 | k8s client-go — HTTP/2 transport |
| `golang.org/x/oauth2` | v0.10.0 | k8s client-go — GKE token refresh |
| `google.golang.org/protobuf` | v1.33.0 | k8s client-go — protobuf serialisation |
| `k8s.io/klog/v2` | v2.110.1 | k8s client-go — structured logging |
| `gopkg.in/yaml.v2` | v2.4.0 | k8s apimachinery — internal use only |

---

## Standard library usage

| Package | Minimum Go version | Usage |
|---|---|---|
| `log/slog` | 1.21 | Structured logging |
| `io/fs` | 1.16 | Directory walking in doc searcher and secret scanner |
| `net/http` | 1.0 | HTTP health checks |
| `encoding/json` | 1.0 | JSON output formatting |
| `regexp` | 1.0 | SAST and secret pattern matching (RE2 syntax — no lookaheads) |
| `os/exec` | 1.0 | Shelling out to `terraform` and `helm` CLIs |

---

## Adding a dependency

Before adding a new dependency:

1. Check if the standard library already provides what you need. The log analyzer, pattern matcher, and doc searcher use only stdlib — no external packages.

2. If an external package is needed, prefer packages with no transitive dependencies or very few. The AWS SDK is the exception, not the model.

3. Run `go mod tidy` after adding and verify `go.sum` is updated.

4. Add an entry to this file with the version, what it does, and what breaks if it changes.
