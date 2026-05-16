<!--
title:       ANNÁVE CLI — Configuration Reference
description: Every key in limits.yaml and messages.yaml, its default value,
             what it controls, and how to change it.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Configuration

ANNÁVE CLI embeds two YAML files at build time. To change a default, edit the file and rebuild the binary.

```bash
# Edit the config
vim internal/shared/config/limits.yaml

# Rebuild
go build -o annave ./cmd/annave
```

There is no runtime config file or environment variable for these values. The embedded defaults are intentional — a binary deployed to a server behaves identically on every machine without any external configuration.

---

## `internal/shared/config/limits.yaml`

Controls per-module resource limits.

### `log`

```yaml
log:
  max_file_size_mb: 256
  max_lines: 1000000
```

| Key | Default | What it controls |
|---|---|---|
| `max_file_size_mb` | 256 | Maximum log file size accepted. Files larger than this are rejected at input with `ERR_INVALID_INPUT`. |
| `max_lines` | 1,000,000 | Maximum number of lines read from a log file or stdin. Lines beyond this limit are silently truncated. |

**When to change:** If you regularly analyze log files from high-traffic services that produce multi-GB logs in a single rotation, increase `max_file_size_mb`. If analysis is too slow on large files, lower `max_lines` to trade completeness for speed.

---

### `health`

```yaml
health:
  timeout_seconds: 10
  max_targets: 50
```

| Key | Default | What it controls |
|---|---|---|
| `timeout_seconds` | 10 | Maximum per-target timeout. If `--timeout` exceeds this value, it is clamped to this limit. |
| `max_targets` | 50 | Maximum number of targets that can be checked in a single run. Targets beyond this limit are ignored. |

**When to change:** If you check services with high cold-start latency (Lambda, Fargate), increase `timeout_seconds`. If you need to check more than 50 targets in a single command, increase `max_targets`.

---

### `doc`

```yaml
doc:
  max_file_size_mb: 10
  max_results: 50
  max_depth: 10
```

| Key | Default | What it controls |
|---|---|---|
| `max_file_size_mb` | 10 | Files larger than this are skipped during indexing. |
| `max_results` | 50 | Maximum number of search results returned. |
| `max_depth` | 10 | Maximum directory depth walked during index building. |

**When to change:** If your documentation includes very large files (generated API references, large YAML specs), increase `max_file_size_mb`. If you have documentation nested deeper than 10 levels, increase `max_depth` — though this is unusual and often indicates a structure problem.

---

### `cleanup`

```yaml
cleanup:
  timeout_seconds: 30
  max_resources: 1000
```

| Key | Default | What it controls |
|---|---|---|
| `timeout_seconds` | 30 | Context deadline for the entire scan. If the cluster API is slow, this may need to be increased. |
| `max_resources` | 1000 | Total idle resources returned across all types. Resources beyond this cap are silently dropped. |

**When to change:** For large clusters with many namespaces, the default 30-second timeout may be too short. Increase `timeout_seconds` to allow more time for API list calls. If you expect more than 1,000 idle resources, increase `max_resources` — though this usually indicates a hygiene problem worth addressing.

---

## `internal/shared/config/messages.yaml`

Maps error codes to human-readable messages. The Go code uses error codes; the messages here are what the user sees.

```yaml
errors:
  ERR_NOT_IMPLEMENTED: "this module is in development and not yet available"
  ERR_INVALID_INPUT: "invalid input provided"
  ERR_IO_FAILURE: "failed to read or write data"
  ERR_TIMEOUT: "operation timed out"
  ERR_PERMISSION: "permission denied"
  ERR_NOT_FOUND: "resource not found"
  ERR_PARSE_FAILED: "failed to parse input"
```

| Code | Message | When emitted |
|---|---|---|
| `ERR_NOT_IMPLEMENTED` | this module is in development and not yet available | Stub adapter invoked (GCP/Azure cost providers) |
| `ERR_INVALID_INPUT` | invalid input provided | Flag validation failures, unsupported format values |
| `ERR_IO_FAILURE` | failed to read or write data | File open/read errors, filesystem permission errors |
| `ERR_TIMEOUT` | operation timed out | Network or API timeout |
| `ERR_PERMISSION` | permission denied | Kubernetes RBAC denial, file permission error |
| `ERR_NOT_FOUND` | resource not found | Target file or k8s resource does not exist |
| `ERR_PARSE_FAILED` | failed to parse input | Malformed YAML, JSON decode error |

**When to change:** If you want different user-facing wording, edit the message for the relevant code. The code values (`ERR_*`) are stable and referenced in Go source — do not change them. Only the message strings are safe to edit.

---

## Build-time vs runtime configuration

There is no runtime configuration mechanism by design. The decision mirrors the ANNÁVE PDF Engine approach: a binary that behaves identically everywhere is easier to reason about and audit than one that reads from environment variables or config files at startup.

If you need different limits for different environments (tighter limits in CI, more generous limits locally), build two binaries with different YAML values. The build is fast (`go build` takes under 5 seconds) and the binary is small (~20 MB with all dependencies embedded as static linking is used by default via `CGO_ENABLED=0`).
