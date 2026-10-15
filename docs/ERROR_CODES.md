<!--
title:       tend: Error Codes Reference
description: The error code tend returns, its stage, when it is triggered,
             and how to handle errors in scripts.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Error Codes Reference

tend returns one error code: `ERR_NOT_IMPLEMENTED`. Every other error is a plain message with no code, such as `error: cannot open "app.log": open app.log: no such file or directory`.

**Plain mode (stderr):**
```
error: [runtime/ERR_NOT_IMPLEMENTED] GCP cost analysis is not implemented yet. Use --provider aws.
```

**JSON mode (stdout, when `--format json` is set and an error occurs):**
```json
{"error":"[runtime/ERR_NOT_IMPLEMENTED] GCP cost analysis is not implemented yet. Use --provider aws."}
```

A coded error has the form `[<stage>/<code>] <message>`. The code is the machine-readable part. The message is for people and may change between versions. Match the code, not the text.

---

## Error codes

### `ERR_NOT_IMPLEMENTED`

| Field | Value |
|---|---|
| Stage | `runtime` |
| Exit code | `1` |

**When triggered:** `tend cost scan --provider gcp` and `tend cost scan --provider azure`. These providers are accepted by the flag but have no working adapter yet.

**Script handling:** Use `--provider aws`.

---

## Errors without a code

Flag, argument, file, network, Kubernetes, AWS, and subprocess errors carry no code. They print one `error:` line to stderr, or `{"error":"..."}` to stdout in JSON mode. Flag, argument, and unknown command errors add one line to stderr that points at `--help`. All of them exit `1`.

---

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | The command ran and found nothing to report |
| `1` | An error occurred, with or without a code |
| `2` | The command ran and found problems: findings, down targets, idle resources, anomalies, or a failed validation |

This means `tend security audit .` returning 10 findings exits `2`, so a CI step fails without parsing the output.
