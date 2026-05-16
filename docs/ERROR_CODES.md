<!--
title:       ANNÁVE CLI — Error Codes Reference
description: All AnnaveError codes, their stage, when each is triggered,
             and how to handle them in scripts.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Error Codes Reference

All errors from ANNÁVE CLI use the same structured format:

**Plain mode (stderr):**
```
[runtime/ERR_NOT_IMPLEMENTED] GCP cost analysis is not yet implemented — use --provider aws
```

**JSON mode (when `--format json` is set and an error occurs):**
```json
{
  "code": "ERR_NOT_IMPLEMENTED",
  "stage": "runtime",
  "message": "GCP cost analysis is not yet implemented — use --provider aws"
}
```

The `code` field is the machine-readable identifier. The `message` field is human-readable and may change between versions. Parse `code`, not `message`.

---

## Code pattern

```
ERR_<SLUG>
```

There are no sequence numbers. The slug is the unique identifier across all modules.

---

## Error codes

### `ERR_NOT_IMPLEMENTED`

| Field | Value |
|---|---|
| Stage | `runtime` |

**When triggered:** A feature that exists in the CLI but has no working adapter yet. Currently: `annave cost scan --provider gcp` and `--provider azure`.

**Script handling:** The exit code is `0` for not-implemented providers — they print a friendly notice, not an error. You only see this code in JSON output or when checking the error type programmatically.

---

### `ERR_INVALID_INPUT`

| Field | Value |
|---|---|
| Stage | `input` |

**When triggered:** A flag value is invalid (unparseable `--since` date, unrecognised `--format` or `--type` value), or a required argument is missing.

**Script handling:** Fix the flag value. The message always includes the invalid value and the accepted alternatives.

---

### `ERR_IO_FAILURE`

| Field | Value |
|---|---|
| Stage | `input` |

**When triggered:** A file cannot be opened or read. Common causes: path does not exist, permission denied, file is a directory when a file is expected.

**Script handling:** Check the file path and permissions. The message includes the OS error string.

---

### `ERR_TIMEOUT`

| Field | Value |
|---|---|
| Stage | `network` |

**When triggered:** A network operation exceeded its deadline. In health checks, each target has its own timeout and a timeout result does not cause a non-zero exit code — it is reported as status `timeout` in the output. This code is used for module-level timeouts (e.g., the entire cleanup scan exceeds `cleanup.timeout_seconds`).

**Script handling:** Increase the timeout in `limits.yaml` or the `--timeout` flag. For cleanup scans on large clusters, increase `cleanup.timeout_seconds`.

---

### `ERR_PERMISSION`

| Field | Value |
|---|---|
| Stage | `runtime` |

**When triggered:** Kubernetes RBAC denied the list or get request. The service account or kubeconfig user does not have permission to list the requested resource type in the requested namespace.

**Script handling:** Check the RBAC policy for the kubeconfig user. At minimum, cleanup scan and security audit require `list` and `get` on pods, pvcs, configmaps, namespaces, and deployments.

---

### `ERR_NOT_FOUND`

| Field | Value |
|---|---|
| Stage | `input` |

**When triggered:** A specified file, directory, or Kubernetes resource does not exist.

**Script handling:** Verify the path or resource name. For Kubernetes contexts, check `kubectl config get-contexts` to confirm the context name exists in the kubeconfig.

---

### `ERR_PARSE_FAILED`

| Field | Value |
|---|---|
| Stage | `analysis` |

**When triggered:** The input cannot be parsed. Common causes: malformed YAML in a Kubernetes manifest, a `terraform show -json` output that does not match the expected schema, or a log file with no lines matching any supported format.

**Script handling:** Verify the input file is valid for its type. For YAML files, run `yaml lint` or `kubectl apply --dry-run=client` to check syntax. For Terraform JSON, ensure the file was produced by `terraform show -json plan.tfplan` and not by another tool.

---

## Stage values

| Stage | Meaning |
|---|---|
| `input` | Error occurred before processing started — bad flag, missing file, invalid format |
| `analysis` | Error occurred during the analysis or parsing phase |
| `output` | Error occurred while formatting or writing the result |
| `runtime` | Error from the adapter at runtime — API call, subprocess, network |
| `network` | Network-specific error — timeout, connection refused, DNS failure |
| `index` | Error during index building (doc search) |

---

## Summary table

| Code | Stage | Exit code |
|---|---|---|
| `ERR_NOT_IMPLEMENTED` | runtime | 0 (printed as notice) |
| `ERR_INVALID_INPUT` | input | 1 |
| `ERR_IO_FAILURE` | input | 1 |
| `ERR_TIMEOUT` | network | 1 |
| `ERR_PERMISSION` | runtime | 1 |
| `ERR_NOT_FOUND` | input | 1 |
| `ERR_PARSE_FAILED` | analysis | 1 |

Note: a command that runs successfully but finds anomalies, issues, or idle resources exits with code `0`. Only execution errors exit with code `1`. This means `annave security audit .` returning 10 findings still exits `0` — treat the findings as data, not as failure.
