<!--
title:       tend: Configuration Reference
description: Every key in limits.yaml, its default value,
             what it controls, and how to change it.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Configuration

tend embeds one YAML file, `limits.yaml`, at build time. To change a default, edit the file and rebuild the binary.

```bash
# Edit the config
vim internal/shared/config/limits.yaml

# Rebuild
go build -o tend ./cmd/tend
```

There is no runtime config file or environment variable for these values. The embedded defaults are intentional: a binary deployed to a server behaves identically on every machine without any external configuration.

---

## `internal/shared/config/limits.yaml`

Controls per-module resource limits.

### `log`

```yaml
log:
  max_lines: 1000000
```

| Key | Default | What it controls |
|---|---|---|
| `max_lines` | 1,000,000 | Maximum number of lines read from a log file or stdin. Reading stops at this limit without a message, and later lines are not analyzed. |

There is no file size limit. Lines longer than 1 MiB are cut to their first 1 MiB; this length is fixed in the code.

**When to change:** If analysis is too slow or uses too much memory on large files, lower `max_lines` to trade completeness for speed. If large logs end before the part you need, raise it.

---

### `health`

```yaml
health:
  timeout_seconds: 10
  max_targets: 50
```

| Key | Default | What it controls |
|---|---|---|
| `timeout_seconds` | 10 | Maximum per-target timeout. If `--timeout` exceeds this value, it is lowered to this limit. |
| `max_targets` | 50 | Maximum number of targets checked at the same time without `--chain`. Every target is checked; the rest wait for a free slot. |

**When to change:** If you check services with high cold-start latency (Lambda, Fargate), increase `timeout_seconds`. If you check many targets and want them all to run at once, increase `max_targets`.

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
| `max_depth` | 10 | Directories more than this many levels below the search root are skipped. |

**When to change:** If your documentation includes very large files (generated API references, large YAML specs), increase `max_file_size_mb`. If you have documentation nested deeper than 10 levels, increase `max_depth`. This is unusual and often indicates a structure problem.

---

### `cleanup`

```yaml
cleanup:
  timeout_seconds: 30
  max_resources: 1000
```

| Key | Default | What it controls |
|---|---|---|
| `timeout_seconds` | 30 | Deadline for the entire `tend cleanup scan`. If the cluster API is slow, this may need to be increased. |
| `max_resources` | 1000 | Total idle resources returned across all types. Resources beyond this cap are silently dropped. |

**When to change:** For large clusters with many namespaces, the default 30-second timeout may be too short. Increase `timeout_seconds` to allow more time for API list calls. If you expect more than 1,000 idle resources, increase `max_resources`. This usually indicates a hygiene problem worth addressing.

---

## Build-time vs runtime configuration

There is no runtime configuration mechanism by design. The decision mirrors the PDF Engine approach: a binary that behaves identically everywhere is easier to reason about and audit than one that reads from environment variables or config files at startup.

If you need different limits for different environments (tighter limits in CI, more generous limits locally), build two binaries with different YAML values. Release builds set `CGO_ENABLED=0` and `-ldflags "-s -w"`, which gives a static binary of about 30 MB on darwin/arm64.
