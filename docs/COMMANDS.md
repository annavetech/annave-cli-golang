<!--
title:       ANNÁVE CLI — Command Reference
description: Full reference for every command, every flag, output format examples,
             and exit codes.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Command Reference

All commands follow the same conventions:

- `--format plain|json|table` is available on every command. Default is `plain`.
- Exit code `0` means the command ran successfully. Exit code `1` means an error occurred. A finding or anomaly is not an error — the command still exits `0`.
- Errors are printed to stderr in plain mode. In JSON mode, errors are printed as `{"code":"...","stage":"...","message":"..."}`.

---

## `annave version`

Print the binary version.

```
annave version
```

Output:
```
annave 0.1.0
```

---

## `annave log analyze`

```
annave log analyze [file] [flags]
```

Analyze a log file for errors, spikes, and repeated patterns.

**Arguments:**
- `file` — path to the log file. Omit when using `--stdin`.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--stdin` | bool | false | read from stdin instead of a file |
| `--format` | string | plain | output format: plain, json, table |
| `--since` | string | — | only include entries after this time |
| `--level` | string | — | minimum log level: debug, info, warn, error |

`--since` accepts a Go duration (`1h`, `30m`, `2h30m`), an RFC3339 timestamp (`2026-05-16T10:00:00Z`), or a date (`2026-05-16`).

**Detected log formats:** JSON structured (any key set including `level`/`msg` or `severity`/`message`), nginx access log, syslog, plain text.

**What it detects:**
- Repeated error patterns — messages that appear 3 or more times
- Time spikes — minutes with 3× the rolling average error rate
- Message clusters — normalised duplicates grouped by template

**Plain output:**
```
  Log analysis — /var/log/app.log
  format          json
  lines parsed    48231 / 48231
  time range      2026-05-15 08:00:01 → 2026-05-16 07:59:58

  3 finding(s):

  [1] CRITICAL  Spike detected at 2026-05-15 14:32 — 847 errors in 1 minute (avg 12/min)
           top message: connection refused: redis:6379
  [2] HIGH     Repeated pattern (124×): failed to acquire database lock
  [3] MEDIUM   Message cluster (38 variants): timeout after [N]ms waiting for [TOKEN]
```

**JSON output shape:**
```json
{
  "file": "/var/log/app.log",
  "format": "json",
  "total_lines": 48231,
  "parsed_lines": 48231,
  "time_range": { "from": "...", "to": "..." },
  "findings": [
    {
      "rank": 1,
      "severity": "critical",
      "summary": "Spike detected at 2026-05-15 14:32 — 847 errors in 1 minute",
      "detail": "top message: connection refused: redis:6379",
      "count": 847
    }
  ]
}
```

---

## `annave health check`

```
annave health check [targets...] [flags]
```

Check uptime and latency of one or more services.

**Arguments:**
- `targets` — one or more target URIs (required).

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--timeout` | string | 10s | per-target timeout |
| `--format` | string | plain | output format: plain, json, table |
| `--chain` | bool | false | stop at first failure (dependency chain mode) |

**Target auto-detection:**

| Input | Check type |
|---|---|
| `https://...` or `http://...` | HTTP — expects 2xx status |
| `tcp://host:port` | TCP port connectivity |
| `dns://host` | DNS resolution |
| `host:port` | TCP (auto-detected) |
| `hostname` | DNS (auto-detected) |

**Plain output:**
```
  Health check — 2026-05-16 10:42:07
  summary         3 total  2 up  1 down

  ✓  https://api.example.com                    UP        200 OK  142ms
  ✓  tcp://redis:6379                           UP                 3ms
  ✗  dns://internal.corp                        TIMEOUT   i/o timeout
```

**JSON output shape:**
```json
{
  "checked_at": "2026-05-16T10:42:07Z",
  "total": 3,
  "up": 2,
  "down": 1,
  "results": [
    {
      "target": { "name": "https://api.example.com", "check_type": "http", "timeout_ns": 10000000000 },
      "status": "up",
      "latency_ns": 142000000,
      "message": "200 OK"
    }
  ]
}
```

---

## `annave doc search`

```
annave doc search [query] [flags]
```

Full-text search across local documentation files.

**Arguments:**
- `query` — search terms (required). Multiple words are treated as AND terms.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-p`, `--path` | string | . | root directory to search |
| `--format` | string | plain | output format: plain, json, table |
| `--open` | bool | false | open the top result in `$EDITOR` |
| `--ext` | []string | — | file extensions to include (repeatable) |

Default extensions: `.md`, `.mdx`, `.txt`, `.rst`, `.html`, `.yaml`, `.json`.

`--open` is line-aware for `nvim`, `vim`, `code` (VS Code), and `nano`. For other editors it opens the file without a line argument.

**Plain output:**
```
  Doc search — "rate limiting"  (3 result(s))

  [1] docs/api-reference.md
       API Reference
       ...requests exceeding the rate limit receive a 429 response...
       line 87     score 1.0  md

  [2] docs/configuration.md
       Configuration Guide
       ...rate_limit.requests_per_minute controls the per-IP bucket...
       line 142    score 0.5  md
```

**JSON output shape:**
```json
[
  {
    "file": {
      "path": "/absolute/path/docs/api-reference.md",
      "rel_path": "docs/api-reference.md",
      "title": "API Reference",
      "format": "md",
      "size_bytes": 14200,
      "mod_time": "2026-05-14T09:00:00Z"
    },
    "line": 87,
    "score": 1.0,
    "excerpt": "...requests exceeding the rate limit receive a 429 response..."
  }
]
```

---

## `annave cleanup scan`

```
annave cleanup scan [flags]
```

Scan for idle Kubernetes resources.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-n`, `--namespace` | string | — | limit scan to a specific namespace |
| `--context` | string | — | kubeconfig context to use |
| `--kubeconfig` | string | — | path to kubeconfig file |
| `--dry-run` | bool | true | preview only, make no changes |
| `--format` | string | plain | output format: plain, json, table |

Kubeconfig resolution: `--kubeconfig` flag → `$KUBECONFIG` environment variable → `~/.kube/config`.

**What it finds:**

| Resource | Condition |
|---|---|
| Pod | Status `Succeeded` (completed job) |
| Pod | Status `Failed` |
| Pod | Status `Running` + container in `CrashLoopBackOff` |
| Pod | Status `Pending` for longer than 10 minutes |
| PVC | Phase `Lost` |
| PVC | Phase `Pending` for longer than 1 hour |
| PVC | Phase `Bound` but not mounted by any pod for over 24 hours |
| ConfigMap | No owner references, older than 30 days |
| Namespace | No pods, deployments, StatefulSets, or services |

**Plain output:**
```
  Cleanup scan [DRY RUN] — 2026-05-16 10:42:07
  context         docker-desktop
  namespace       all namespaces
  findings        5 idle resource(s)

  Pod (3):
    production/worker-job-4x9mt                       Completed             finished 3d ago
    staging/api-6b8f9d-abcde                          CrashLoopBackOff      14 restarts
    staging/migration-job-1a2b3                        Failed                exit code 1

  PVC (2):
    production/data-postgres-0                         BoundUnmounted        unmounted 48h
    staging/logs-collector-pvc                         Pending               pending 2h15m
```

**JSON output shape:**
```json
{
  "context": "docker-desktop",
  "namespace": "",
  "dry_run": true,
  "scanned_at": "2026-05-16T10:42:07Z",
  "resources": [
    {
      "resource": {
        "kind": "Pod",
        "name": "worker-job-4x9mt",
        "namespace": "production",
        "age_hours": 72
      },
      "reason": "Completed",
      "detail": "finished 3d ago"
    }
  ]
}
```

---

## `annave security audit`

```
annave security audit [path] [flags]
```

Audit a path for security issues.

**Arguments:**
- `path` — directory or file to audit (default: `.`).

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--type` | string | secrets | scan type: secrets, sast, k8s-live, k8s-local |
| `--format` | string | plain | output format: plain, json, table |
| `--kubeconfig` | string | — | kubeconfig file (k8s-live only) |
| `--context` | string | — | kubeconfig context (k8s-live only) |

**Scan types:**

`secrets` — 12 patterns: `SECRET001` AWS access key, `SECRET002` AWS secret key, `SECRET003` GCP service account JSON, `SECRET004` RSA private key, `SECRET005` EC private key, `SECRET006` generic PEM private key, `SECRET007` JWT secret assignment, `SECRET008` generic API token/key, `SECRET009` database URL with password, `SECRET010` GitHub token, `SECRET011` Slack token, `SECRET012` generic password assignment.

`sast` (Go) — 10 rules: `GO001` command injection, `GO002` SQL injection, `GO003` path traversal, `GO004` SSRF, `GO005` weak random, `GO006` weak crypto (MD5), `GO007` weak crypto (SHA1), `GO008` plaintext HTTP server, `GO009` unbounded read, `GO010` unsafe import, `GO010` hardcoded credential.

`sast` (TypeScript) — 5 rules: `TS001` innerHTML XSS, `TS002` dangerouslySetInnerHTML, `TS003` eval, `TS004` document.write, `TS005` sensitive localStorage.

`k8s-live` — 7 rules: `K8S001` root container, `K8S002` no CPU limit, `K8S003` no memory limit, `K8S004` privileged container, `K8S005` hostPath volume, `K8S006` missing readiness probe, `K8S007` hostNetwork or hostPID.

`k8s-local` — same 7 rules plus `K8S008` unpinned image tag (`:latest` or no tag).

**Plain output:**
```
  Security audit — . [secrets]
  scanned at      2026-05-16 10:42:07
  findings        2
    high          1
    medium        1

  [1] HIGH     SECRET001  AWS Access Key
       file: ./config/deploy.sh:14
       detail: AKIA****************XAMPLE
       fix: Move to environment variable or AWS Secrets Manager

  [2] MEDIUM   SECRET012  Password in source
       file: ./.env.backup:3
       detail: password=pa**...
       fix: Remove from source; use a secrets manager
```

**JSON output shape:**
```json
{
  "target": { "path": ".", "type": "secrets" },
  "scanned_at": "2026-05-16T10:42:07Z",
  "findings": [
    {
      "id": "SECRET001",
      "title": "AWS Access Key",
      "severity": "high",
      "file": "./config/deploy.sh",
      "line": 14,
      "detail": "AKIA****************XAMPLE",
      "remediation": "Move to environment variable or AWS Secrets Manager"
    }
  ],
  "summary": { "high": 1, "medium": 1 }
}
```

---

## `annave infra validate`

```
annave infra validate [target] [flags]
```

Validate infrastructure definitions.

**Arguments:**
- `target` — path to a file or directory (optional; defaults depend on `--type`).

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--type` | string | auto | validate type: terraform, helm, k8s |
| `--format` | string | plain | output format: plain, json, table |

**Auto-detection:**

| Target | Detected as |
|---|---|
| `*.json`, `*.tfplan` | terraform |
| directory with `Chart.yaml` | helm chart lint |
| `*.yaml`, `*.yml`, directory | Kubernetes manifests |
| no target | helm release list (`helm list -A`) |

**Terraform rules:** `TF001` destructive change (delete/replace), `TF002` IAM resource modification, `TF003` network/firewall change, `TF004` data resource deletion risk, `TF005` database resource modification.

**Helm rules:** `HELM001` release in Failed state, `HELM002` release in Pending state, `HELM003` chart lint error, `HELM004` chart lint warning, `HELM005` chart lint info.

**Kubernetes rules:** `K8S101` deprecated apiVersion, `K8S102` missing metadata.name, `K8S103` image tag is `latest` or missing, `K8S104` Deployment with replicas:1 and no PodDisruptionBudget, `K8S105` missing resource limits, `K8S106` missing liveness probe.

**Plain output:**
```
  Infra validation — plan.json
  validated at    2026-05-16 10:42:07
  result          FAILED (3 issue(s))

  [1] CRITICAL  TF001  Destructive change: aws_db_instance.production (replace)
  [2] HIGH      TF002  IAM modification: aws_iam_role.app_role (update)
  [3] MEDIUM    TF003  Security group change: aws_security_group.web (update)
```

**JSON output shape:**
```json
{
  "target": "plan.json",
  "validated_at": "2026-05-16T10:42:07Z",
  "passed": false,
  "issues": [
    {
      "rule": "TF001",
      "severity": "critical",
      "message": "Destructive change: aws_db_instance.production (replace)",
      "resource": "aws_db_instance.production"
    }
  ]
}
```

---

## `annave cost scan`

```
annave cost scan [flags]
```

Fetch cloud costs and detect anomalies.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--provider` | string | aws | cloud provider: aws, gcp, azure |
| `--since` | string | 30 days ago | billing period start date (YYYY-MM-DD) |
| `--format` | string | plain | output format: plain, json, table |

**AWS authentication** — standard credential chain: `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` environment variables → `~/.aws/credentials` → IAM instance role. The IAM policy requires `ce:GetCostAndUsage` and `ce:GetCostForecast`. Cost Explorer charges $0.01 per API request.

**Anomaly detection** — a service is flagged if its cost on the last day of the period is more than 20% above its 7-day rolling average AND the absolute increase exceeds $5. Both conditions must be met to filter noise.

**Forecast** — a 30-day projected cost is fetched via `GetCostForecast` and displayed if available. For new accounts or accounts with less than 14 days of data, the forecast API returns an error and the forecast is omitted silently.

**Plain output:**
```
  Cost analysis — AWS
  period          2026-04-16 → 2026-05-15
  total cost      $1,284.72 USD
  scanned at      2026-05-16 10:42:07
  forecast (next 30 days)  $1,310.00 USD

  TOP SERVICES BY COST (12):
    Amazon EC2                                         $  612.40  ( 47.7%)
    Amazon RDS                                         $  287.15  ( 22.4%)
    Amazon S3                                          $  156.88  ( 12.2%)
    AWS Lambda                                         $   88.42  (  6.9%)
    Amazon CloudFront                                  $   62.17  (  4.8%)
    ...and 7 more services

  ANOMALIES (1):
  [1] Amazon RDS — 34% above 7-day average ($28.71/day vs $21.43/day avg)
```

**JSON output shape:**
```json
{
  "provider": "aws",
  "period": "2026-04-16 → 2026-05-15",
  "total_cost": 1284.72,
  "currency": "USD",
  "scanned_at": "2026-05-16T10:42:07Z",
  "records": [
    { "service": "Amazon EC2", "amount": 612.40, "currency": "USD", "period": "..." }
  ],
  "anomalies": [
    {
      "service": "Amazon RDS",
      "message": "34% above 7-day average ($28.71/day vs $21.43/day avg)",
      "expected": 21.43,
      "actual": 28.71,
      "delta_pct": 34.0
    }
  ],
  "by_resource": [
    { "resource": "forecast (next 30 days)", "amount": 1310.00, "currency": "USD" }
  ]
}
```
