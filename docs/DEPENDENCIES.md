<!--
title:       tend: Dependencies
description: Every direct dependency, exact pinned version, what each does,
             and what breaks if you change the version.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Dependencies

Go version: **1.26** (minimum, the `go 1.26.0` line in `go.mod`). The toolchain is pinned to go1.26.9 (`toolchain` in `go.mod` and `.tool-versions`).

All dependencies are listed in `go.mod`. Run `go mod verify` to confirm checksums match `go.sum`.

---

## Direct dependencies

### `github.com/spf13/cobra v1.10.2`

CLI framework. Provides command registration, flag parsing, help text generation, and shell completion. The CLI uses `cobra.Command` exclusively, with no custom argument parsing.

| Version range | Result |
|---|---|
| < v1.8.0 | Unknown: `FParseErrWhitelist` may not exist |
| v1.10.2 | Confirmed working |
| v1.11+ | Unknown: the API is stable and should work |

All commands are registered in `init()` blocks using `parent.AddCommand(child)`. Never call `Execute()` more than once.

---

### `gopkg.in/yaml.v3 v3.0.1`

YAML parsing for the embedded `limits.yaml` config file.

| Version range | Result |
|---|---|
| v3.0.1 | Confirmed working |
| v3.0.0 | Unknown |

Do not use `gopkg.in/yaml.v2`. The module graph does not need it. Use v3 for config and `sigs.k8s.io/yaml` for Kubernetes manifests.

---

### `k8s.io/client-go v0.37.1`

Kubernetes client for `tend cleanup scan` and `tend security audit --type k8s-live`. Provides the typed client (`kubernetes.Clientset`), kubeconfig loading (`tools/clientcmd`), and list/watch APIs.

| Version range | Result |
|---|---|
| v0.37.1 | Confirmed working (the `go.mod` version) |
| other versions | Unknown: check that the kubeconfig loading API is unchanged |

This is the largest transitive dependency tree in the project (30+ packages). Updating it should be done carefully and tested against a real cluster.

---

### `k8s.io/api v0.37.1` and `k8s.io/apimachinery v0.37.1`

Kubernetes API type definitions used by client-go. Must be pinned to the same minor version as client-go. Do not update one without the other.

---

### `sigs.k8s.io/yaml v1.6.0`

YAML parsing for Kubernetes manifests in `tend security audit --type k8s-local` and `tend infra validate --type k8s`. This library converts YAML to JSON and decodes it into Go structs through their `json` tags, the same way Kubernetes API types are decoded.

It is a transitive dependency of client-go, so it is always present. Using it directly avoids a separate dependency for manifest parsing.

---

### `github.com/aws/aws-sdk-go-v2 v1.47.3`

AWS SDK v2 core. `tend cost scan` imports its `aws` package for request helpers (`aws.String`, `aws.ToString`).

---

### `github.com/aws/aws-sdk-go-v2/config v1.33.9`

AWS SDK v2 configuration loading for `tend cost scan`. Implements the credential chain: environment variables, then the shared credentials file, then the IAM instance role.

---

### `github.com/aws/aws-sdk-go-v2/service/costexplorer v1.74.1`

AWS Cost Explorer client. Used for two API calls: `GetCostAndUsage` (daily costs by service) and `GetCostForecast` (30-day projected spend). The Cost Explorer API is a global endpoint; it does not require a region to be set in the config.

Note: Cost Explorer charges $0.01 per API request. Each `tend cost scan` call makes 1 or 2 requests (cost data + optional forecast).

---

## Indirect dependencies (selected)

These are required by direct dependencies but not imported directly.

| Package | Version | Required by |
|---|---|---|
| `github.com/aws/smithy-go` | v1.28.5 | aws-sdk-go-v2: HTTP transport and signing |
| `github.com/spf13/pflag` | v1.0.10 | cobra: POSIX flag parsing (a direct requirement in `go.mod`, imported only by tests) |
| `golang.org/x/net` | v0.61.0 | k8s client-go: HTTP/2 transport |
| `golang.org/x/oauth2` | v0.36.0 | k8s client-go: token refresh |
| `google.golang.org/protobuf` | v1.36.12-0.20260120151049-f2248ac996af | k8s client-go: protobuf serialisation |
| `k8s.io/klog/v2` | v2.140.0 | k8s client-go: structured logging |

---

## Standard library usage

| Package | Minimum Go version | Usage |
|---|---|---|
| `io/fs` | 1.16 | Directory walking in doc searcher and secret scanner |
| `net/http` | 1.0 | HTTP health checks |
| `encoding/json` | 1.0 | JSON output formatting |
| `regexp` | 1.0 | SAST and secret pattern matching (RE2 syntax, no lookaheads) |
| `os/exec` | 1.0 | Shelling out to `terraform` and `helm` CLIs |

---

## Adding a dependency

Before adding a new dependency:

1. Check if the standard library already provides what you need. The log analyzer, pattern matcher, and doc searcher use only the standard library and no external packages.

2. If an external package is needed, prefer packages with no transitive dependencies or very few. The AWS SDK is the exception, not the model.

3. Run `go mod tidy` after adding and verify `go.sum` is updated.

4. Add an entry to this file with the version, what it does, and what breaks if it changes.
