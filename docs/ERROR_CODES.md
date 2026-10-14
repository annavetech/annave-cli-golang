<!--
title:       tend: Error Codes Reference
description: All tend error codes, their stage, when each is triggered,
             and how to handle them in scripts.
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Error Codes Reference

All errors from tend use the same structured format:

**Plain mode (stderr):**
```
[runtime/ERR_NOT_IMPLEMENTED] GCP cost analysis is not implemented yet. Use --provider aws.
```

**JSON mode (stdout, when `--format json` is set and an error occurs):**
```json
{"error":"[runtime/ERR_NOT_IMPLEMENTED] GCP cost analysis is not implemented yet. Use --provider aws."}
```

The `ERR_<SLUG>` code inside the brackets is the machine-readable identifier. The rest of the message is human-readable and may change between versions. Match the code, not the text.

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

**When triggered:** A feature that exists in the CLI but has no working adapter yet. Currently: `tend cost scan --provider gcp` and `--provider azure`.

**Script handling:** The command exits `1`. Use `--provider aws` instead.

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

**When triggered:** A network operation exceeded its deadline. In health checks, each target has its own timeout and a timed out target is reported as status `timeout` in the output; the command then exits `2`, like any down target. This code is used for module-level timeouts (e.g., the entire cleanup scan exceeds `cleanup.timeout_seconds`).

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
| `input` | Error occurred before processing started: bad flag, missing file, invalid format |
| `analysis` | Error occurred during the analysis or parsing phase |
| `output` | Error occurred while formatting or writing the result |
| `runtime` | Error from the adapter at runtime: API call, subprocess, network |
| `network` | Network-specific error: timeout, connection refused, DNS failure |
| `index` | Error during index building (doc search) |

---

## Summary table

| Code | Stage | Exit code |
|---|---|---|
| `ERR_NOT_IMPLEMENTED` | runtime | 1 |
| `ERR_INVALID_INPUT` | input | 1 |
| `ERR_IO_FAILURE` | input | 1 |
| `ERR_TIMEOUT` | network | 1 |
| `ERR_PERMISSION` | runtime | 1 |
| `ERR_NOT_FOUND` | input | 1 |
| `ERR_PARSE_FAILED` | analysis | 1 |

Note: a command that runs successfully but reports findings, down targets, anomalies, idle resources, or a failed validation exits with code `2`. Execution errors exit with code `1`. A clean run exits `0`. This means `tend security audit .` returning 10 findings exits `2`, so a CI step fails without parsing the output.
