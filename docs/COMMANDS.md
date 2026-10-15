<!--
title:       tend: Command Reference
description: Full reference for every command, every flag, output format examples,
             and exit codes.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Command Reference

All commands follow the same conventions:

- `--format plain|json|table` is available on every command. Default is `plain`.
- `--format` values are case-insensitive. Any other value is an error.
- Exit code `0` means the command ran and found nothing to report. Exit code `1` means an error occurred. Exit code `2` means the command ran and found problems: findings, down targets, idle resources, anomalies, or a failed validation. Each command lists its exit codes below.
- Errors are printed to stderr as one `error:` line. Flag, argument, and unknown command errors add one line that points at `--help` (`Run 'tend log analyze --help' for usage.`). In JSON mode, errors are printed to stdout as `{"error":"..."}`, and notices go to stderr so stdout stays valid JSON.

---

## `tend version`

Print the binary version.

```
tend version
```

Output:
```
tend version 0.3.0
```

A release build prints the tag version. A `go install` build prints the module version. A `go build` in a git checkout prints a commit-based version such as `0.0.0-20261014154829-6106e817b6cb`, with `+dirty` when the tree has uncommitted changes.

---

## `tend log analyze`

```
tend log analyze [file] [flags]
```

Analyze a log file for errors, spikes, and repeated patterns.

**Arguments:**
- `file`: path to the log file, or `-` to read from stdin. Required unless `--stdin` is set.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--stdin` | bool | false | read from stdin instead of a file |
| `--format` | string | plain | output format: plain, json, table |
| `--since` | string | none | only include entries at or after this time |
| `--level` | string | none | minimum log level: debug, info, warn, error, critical (`warning` is accepted as warn; any other value is an error) |

`--since` accepts a Go duration (`1h`, `30m`, `2h30m`), an RFC3339 timestamp (`2026-05-16T10:00:00Z`), or a date (`2026-05-16`). Entries without a parsed timestamp are always kept.

`--level` keeps entries at or above the level. The order is debug, info, warn, error, critical.

**Detected log formats:** JSON structured, nginx access log, BSD syslog, plain text. The format is detected from the first non-blank line and used for every line.

For JSON lines, the timestamp is read from `time`, `timestamp`, `ts`, or `@timestamp`, the level from `level`, `severity`, `lvl`, or `loglevel`, and the message from `msg`, `message`, `text`, or `body`. For nginx lines, status 5xx is error, 4xx is warn, and anything else is info. For syslog and plain text, the level is the first level keyword found in the line, and info when there is none.

At most 1,000,000 lines are read (`log.max_lines` in `limits.yaml`).

Lines longer than 1 MiB are cut to their first 1 MiB and reading continues. The count is shown as `truncated lines` in plain output and `truncated_lines` in JSON output. A read error ends the command with an error.

**What it detects:**
- Repeated patterns: the same warn, error, or critical message 3 or more times.
- Spikes: needs at least 5 error or critical entries. With timestamps, a one-minute window with at least 5 errors and at least 3x the average errors per minute (the average over minutes that have errors). Without timestamps, the same test runs on line windows, each a tenth of the analysed entries and at least 10 lines long.
- Clusters: warn, error, or critical messages that become the same template after UUIDs, long hex strings, IPv4 addresses, quoted strings of 8 or more characters, and whole numbers are replaced with `<uuid>`, `<hex>`, `<ip>`, `<value>`, and `<N>`. A cluster needs 3 or more messages. Messages with no variable part are left to the pattern check.

Spikes are listed first, then the rest by count. A spike of 50 or more errors is critical, and other spikes are high. A pattern or cluster is critical at 10% of the analysed entries or 1,000 occurrences, high at 5% or 100, medium at 1% or 20, and low otherwise.

**Plain output:**
```
Log analysis: app.log
---------------------
  format:              json
  lines parsed:        65 / 65
  time range:          2026-05-15 14:00:00 to 2026-05-15 14:13:04

  3 finding(s):

  [1] HIGH      7x error spike at 14:12 (40 errors in 1 minute)
           lines 21 to 60
  [2] CRITICAL  repeated 40 times: connection refused: redis:6379
           lines 21 to 60
           sample: {"time": "2026-05-15T14:12:00Z", "level": "error", "msg": "connection refused: redis:6379"}
  [3] CRITICAL  cluster of 40: connection refused: redis:<N>
           lines 21 to 60
           sample: connection refused: redis:6379
```

`lines parsed` is the number of non-blank lines over the number of lines read. When there are no findings, the output ends with `No anomalies detected.`

**JSON output shape:**
```json
{
  "file": "app.log",
  "format": "json",
  "total_lines": 65,
  "parsed_lines": 65,
  "time_range": { "from": "2026-05-15T14:00:00Z", "to": "2026-05-15T14:13:04Z" },
  "findings": [
    {
      "rank": 1,
      "severity": "high",
      "summary": "7x error spike at 14:12 (40 errors in 1 minute)",
      "detail": "lines 21 to 60",
      "count": 40,
      "anomaly": {
        "kind": "spike",
        "message": "7x error spike at 14:12 (40 errors in 1 minute)",
        "count": 40,
        "first_line": 21,
        "last_line": 60,
        "first_seen": "2026-05-15T14:12:00Z",
        "last_seen": "2026-05-15T14:12:39Z"
      }
    }
  ]
}
```

`severity` is `critical`, `high`, `medium`, or `low`. `anomaly.kind` is `spike`, `pattern`, or `cluster`. Pattern and cluster anomalies also carry `sample_lines` (up to 3). `parse_errors` and `truncated_lines` appear only when they are not zero. `findings` is `null` when there are none.

**Exit codes:** `0` no findings, `1` error, `2` one or more findings.

---

## `tend health check`

```
tend health check [targets...] [flags]
```

Check uptime and latency of one or more services.

**Arguments:**
- `targets`: one or more target URIs (required).

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--timeout` | string | 10s | per-target timeout, a Go duration; values above 10s are lowered to 10s (`health.timeout_seconds`) |
| `--format` | string | plain | output format: plain, json, table |
| `--chain` | bool | false | check targets one by one in the given order and stop at the first target that is not up |

**Target auto-detection:**

| Input | Check type |
|---|---|
| `https://...` or `http://...` | HTTP GET (expects a 2xx status, follows up to 4 redirects) |
| `tcp://host:port` | TCP port connectivity |
| `dns://host` | DNS resolution with the system resolver |
| any other input containing `:` | TCP (auto-detected) |
| any other input | DNS (auto-detected) |

Without `--chain`, all targets are checked at the same time, at most 50 at once (`health.max_targets`). With `--chain`, the report lists only the targets that were checked. The timeout applies to each HTTP request and TCP dial. DNS lookups are bounded by the run deadline: the timeout plus 5 seconds, or in chain mode the timeout times the number of targets plus one.

**Plain output:**
```
Health check: 2026-05-16 10:42:07
---------------------------------
  summary:             3 total  2 up  1 down

  https://api.example.com                        UP        HTTP 200  142ms
  tcp://redis:6379                               UP        TCP redis:6379 open  3ms
  dns://internal.corp                            TIMEOUT   DNS resolution timed out  10s
```

Status is `UP`, `DOWN`, `TIMEOUT`, or `UNKNOWN`. Every status other than `UP` counts as down. Latency is rounded to the millisecond.

**JSON output shape:**
```json
{
  "checked_at": "2026-05-16T10:42:07Z",
  "results": [
    {
      "target": {
        "name": "https://api.example.com",
        "address": "https://api.example.com",
        "check_type": "http",
        "expect_status": 0,
        "expect_body": "",
        "timeout_ms": 10000
      },
      "status": "up",
      "message": "HTTP 200",
      "checked_at": "2026-05-16T10:42:07Z",
      "latency_ms": 142.031
    }
  ],
  "up": 2,
  "down": 1,
  "total": 3
}
```

`target.timeout_ms` and `latency_ms` are in milliseconds, as decimal numbers. `check_type` is `http`, `tcp`, or `dns`. `status` is `up`, `down`, `timeout`, or `unknown`. `expect_status` and `expect_body` cannot be set from the command line and are always `0` and `""`.

**Exit codes:** `0` all targets up, `1` error, `2` one or more targets down or timed out.

---

## `tend doc search`

```
tend doc search [query] [flags]
```

Full-text search across local documentation files.

**Arguments:**
- `query`: search terms (required). All arguments are joined into one query and split into words.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-p`, `--path` | string | . | root directory to search |
| `--format` | string | plain | output format: plain, json, table |
| `--open` | bool | false | open the top result in an editor instead of printing results |
| `--ext` | string, repeatable | none | only search files with this extension, with or without the dot (`--ext md --ext txt`) |

Searched extensions: `.md`, `.mdx`, `.txt`, `.rst`, `.html`, `.htm`, `.yaml`, `.yml`, `.json`. `--ext` can only narrow this list.

Hidden directories and `node_modules`, `vendor`, `build`, `dist`, `__pycache__`, `.next`, `.angular`, `.idea`, `.vscode`, `.cache`, `.git`, `.svn` are skipped. Empty files, files over 10 MB (`doc.max_file_size_mb`), and directories more than 10 levels below `--path` (`doc.max_depth`) are skipped. At most 50 results are returned (`doc.max_results`).

**Scoring:** words are split on any character other than letters, digits, `-`, and `_`, lowercased, and kept when 3 or more characters long. For each query word, every line that contains it as a whole word adds 1.0 to its file. Only when no file contains the whole word, every line with a word that starts with it adds 0.5. A file matches when any query word matches. Results are sorted by score, then by most recently modified. The line shown is the first matching line, and the title is the first heading or the file name.

`--open` uses the first word of `$EDITOR`, then `$VISUAL`, as the program and the remaining words as arguments. Without either, it uses the first of `code`, `nvim`, `vim`, `nano` found on `PATH`, and `vi` otherwise. The line is passed to `vim`, `nvim`, `nano` (`+N`) and `code` (`--goto file:N`). In JSON mode the `Opening:` notice goes to stderr.

**Plain output:**
```
Doc search: "rate limit"  (1 result(s))
---------------------------------------

  [1] api-reference.md
       API Reference
       Requests exceeding the rate limit receive a 429 response.
       line 5      score 2.0  md
```

Paths are relative to `--path`. With no results, the output is one line: `No results for "rate limit" in docs`.

**JSON output shape:**
```json
[
  {
    "file": {
      "path": "/absolute/path/docs/api-reference.md",
      "rel_path": "api-reference.md",
      "format": "md",
      "title": "API Reference",
      "size_bytes": 83,
      "mod_time": "2026-05-14T09:00:00Z"
    },
    "score": 2,
    "excerpt": "Requests exceeding the rate limit receive a 429 response.",
    "line": 5
  }
]
```

`format` is `md`, `txt`, `rst`, `html`, `yaml`, or `json`. The excerpt is the matching line, trimmed and cut to 200 characters.

**Exit codes:** `0` on success, including no results (JSON mode prints `[]`), `1` error.

---

## `tend cleanup scan`

```
tend cleanup scan [flags]
```

Scan for idle Kubernetes resources.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-n`, `--namespace` | string | all namespaces | limit scan to a specific namespace |
| `--context` | string | none | kubeconfig context to use |
| `--kubeconfig` | string | none | path to kubeconfig file |
| `--dry-run` | bool | true | only sets the `[DRY RUN]` label and the `dry_run` key; the command never changes or deletes anything |
| `--format` | string | plain | output format: plain, json, table |

Kubeconfig resolution: the `--kubeconfig` flag, then the `$KUBECONFIG` environment variable, then `~/.kube/config`.

The whole scan has a 30 second deadline (`cleanup.timeout_seconds`), and at most 1,000 resources are listed (`cleanup.max_resources`).

If every resource type fails to scan (for example, the cluster is unreachable), the command prints the errors and exits with code 1. If only some resource types or namespaces fail, the plan lists the rest and one `warning:` line per failure is printed to stderr. JSON output also lists them under `warnings`.

**What it finds:**

| Resource | Condition | Reason |
|---|---|---|
| Pod | Phase `Succeeded` | `Completed` |
| Pod | Phase `Failed` | `Failed` |
| Pod | Phase `Running` and a container waiting in `CrashLoopBackOff` | `CrashLoopBackOff` |
| Pod | Phase `Pending` and created more than 10 minutes ago | `PendingTooLong` |
| PersistentVolumeClaim | Phase `Lost` | `NotBound` |
| PersistentVolumeClaim | Phase `Pending` and created more than 1 hour ago | `NotBound` |
| PersistentVolumeClaim | Phase `Bound`, not used by any pod, and created more than 24 hours ago | `Orphaned` |
| ConfigMap | No owner references and created more than 30 days ago | `Orphaned` |
| Namespace | Created more than 24 hours ago and has no pods, Deployments, StatefulSets, or Services | `Empty` |

Pods and ConfigMaps in `kube-system`, `kube-public`, and `kube-node-lease` are skipped, as is the `kube-root-ca.crt` ConfigMap. PersistentVolumeClaims are checked in every namespace. The namespace check also skips `default`.

**Plain output:**
```
Cleanup scan [DRY RUN]: 2026-05-16 10:42:07
-------------------------------------------
  context:             docker-desktop
  namespace:           all namespaces
  findings:            4 idle resource(s)

  Pod (2):
    production/worker-job-4x9mt                         Completed             job pod completed
    staging/api-6b8f9d-abcde                            CrashLoopBackOff      container api: 14 restarts

  PersistentVolumeClaim (2):
    production/data-postgres-0                          Orphaned              bound but not mounted by any pod
    staging/logs-collector-pvc                          NotBound              pending for 2h15m0s with no matching PV
```

**JSON output shape:**
```json
{
  "context": "docker-desktop",
  "resources": [
    {
      "resource": {
        "kind": "Pod",
        "name": "worker-job-4x9mt",
        "namespace": "production",
        "age_hours": 72,
        "labels": { "job-name": "worker-job" }
      },
      "reason": "Completed",
      "detail": "job pod completed"
    }
  ],
  "scanned_at": "2026-05-16T10:42:07Z",
  "dry_run": true,
  "warnings": ["namespace staging not checked: ..."]
}
```

`resource.kind` is `Pod`, `PersistentVolumeClaim`, `ConfigMap`, or `Namespace`. `reason` is one of the reasons in the table above. `namespace`, `labels`, `detail`, and `warnings` are left out when empty, so `namespace` is absent when the scan covers all namespaces. `resources` is `null` when nothing is found.

**Exit codes:** `0` no idle resources, `1` error, `2` one or more idle resources.

---

## `tend security audit`

```
tend security audit [path] [flags]
```

Audit a path for security issues.

**Arguments:**
- `path`: directory or file to audit (default: `.`).

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--type` | string | secrets | scan type: secrets, sast, k8s-live, k8s-local (case-insensitive) |
| `--format` | string | plain | output format: plain, json, table |
| `--kubeconfig` | string | none | kubeconfig file (k8s-live only; then `$KUBECONFIG`, then `~/.kube/config`) |
| `--context` | string | current context | kubeconfig context (k8s-live only) |
| `--namespace` | string | all namespaces | namespace to scan (k8s-live only) |

**Scan types:**

`secrets` has 12 patterns: `SECRET001` AWS access key ID (critical), `SECRET002` AWS secret access key (critical), `SECRET003` private key header (critical), `SECRET004` GitHub token (high), `SECRET005` Slack token (high), `SECRET006` JWT (high), `SECRET007` database URL with credentials (high), `SECRET008` hardcoded password or secret (high), `SECRET009` generic API key (medium), `SECRET010` Stripe secret key (critical), `SECRET011` SendGrid API key (high), `SECRET012` GCP service account key (critical).

`secrets` scans every file under the path, including hidden files and files listed in `.gitignore`. It skips the directories `node_modules`, `vendor`, `dist`, `build`, `.git`, `.angular`, `__pycache__`, `.next`, `coverage`, `.cache`, and skips symlinks, empty files, files over 5 MB, and files with a zero byte in the first 512 bytes.

`sast` (Go, `.go` files) has 10 rules: `GO001` command injection (critical), `GO002` SQL injection (critical), `GO003` path traversal (high), `GO004` SSRF (high), `GO005` weak random, `math/rand` (medium), `GO006` weak crypto, MD5 or SHA-1 (medium), `GO007` plaintext HTTP server (medium), `GO008` unbounded read, `io.ReadAll` (low), `GO009` `unsafe` import (medium), `GO010` hardcoded credential (high).

`sast` (TypeScript and JavaScript, `.ts`, `.tsx`, `.js`, `.jsx` files) has 5 rules: `TS001` innerHTML assignment (high), `TS002` dangerouslySetInnerHTML (high), `TS003` eval (critical), `TS004` document.write (high), `TS005` sensitive data in localStorage (medium).

`k8s-live` lists pods in `--namespace`, or in all namespaces, and has 7 rules: `K8S001` hostNetwork (high), `K8S002` hostPID (high), `K8S003` hostPath volume (medium), `K8S004` privileged container (critical), `K8S005` container may run as root (high), `K8S006` no resource limits (medium), `K8S007` missing readiness probe (low). The path argument is not used.

`k8s-local` reads `.yaml` and `.yml` files under the path with the same 7 rules plus `K8S008` unpinned image tag, `:latest` or no tag (low). It reads Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet, Job, and CronJob specs.

Both Kubernetes scans check init containers and ephemeral containers as well as regular containers. Readiness probe checks apply only to regular containers, and resource limit checks do not apply to ephemeral containers. The root check reads both the pod-level and the container-level `securityContext`. `k8s-local` reads CronJob pods from `spec.jobTemplate.spec.template.spec`.

`secrets` and `sast` print a `warning:` line to stderr for a file that could not be read to the end. Findings from that file before the error are kept. In `secrets` findings, the detail shows the first 12 characters of the match, then up to 8 `*` and `...`. A match of 12 characters or fewer is shown in full. In `sast` findings, the detail is the trimmed source line, cut to 120 characters. For `GO010` and `TS005`, the detail shows only the match up to its first quote or parenthesis, then `********`, never the full source line.

**Plain output:**
```
Security audit: . [secrets]
---------------------------
  scanned at:          2026-05-16 10:42:07
  findings:            2
    high:              1
    critical:          1

  [1] HIGH      SECRET007  Database URL with credentials
       file: .env.backup:1
       detail: postgres://a********...
       fix: Move database credentials to environment variables or a secrets manager.

  [2] CRITICAL  SECRET001  AWS Access Key ID
       file: config/deploy.sh:14
       detail: AKIAABCDEFGH********...
       fix: Remove the key immediately, rotate it in AWS IAM, and use environment variables or IAM roles.
```

Findings are listed in scan order, not by severity. Kubernetes findings put the object reference in `file`, such as `deploy.yaml[0] (Deployment/web) (container: web)` or `default/web-0 (container: web)`, and have no line.

**JSON output shape:**
```json
{
  "target": { "path": ".", "type": "secrets" },
  "findings": [
    {
      "id": "SECRET001",
      "title": "AWS Access Key ID",
      "severity": "critical",
      "file": "config/deploy.sh",
      "line": 14,
      "detail": "AKIAABCDEFGH********...",
      "remediation": "Remove the key immediately, rotate it in AWS IAM, and use environment variables or IAM roles."
    }
  ],
  "scanned_at": "2026-05-16T10:42:07Z",
  "summary": { "critical": 1, "high": 1 }
}
```

`summary` counts findings per severity and lists only severities that occur. `file`, `line`, `detail`, and `remediation` are left out when empty. `warnings` lists files that were only partly scanned and is left out when empty. `findings` is `null` when there are none.

**Exit codes:** `0` no findings, `1` error, `2` one or more findings.

---

## `tend infra validate`

```
tend infra validate [target] [flags]
```

Validate infrastructure definitions.

**Arguments:**
- `target`: path to a file or directory (optional; defaults depend on `--type`).

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--type` | string | none (auto-detect) | validate type: terraform, helm, k8s |
| `--format` | string | plain | output format: plain, json, table |

**Auto-detection:**

| Target | Detected as |
|---|---|
| `*.json`, `*.tfplan` | terraform |
| directory with `Chart.yaml` | helm chart lint |
| any other file or directory, or a path that does not exist | Kubernetes manifests |
| no target, or `.` | helm release list (`helm list -A -o json --all`) |

A `.json` plan is read directly. A `.tfplan` file is converted with `terraform show -json`, which needs `terraform` on `PATH`. The helm checks need `helm` on `PATH`.

**Result:** the validation FAILS when any issue is medium, high, or critical. Only low and info issues give PASSED.

**Terraform rules:** read from `resource_changes`; `no-op` and `read` actions are ignored.
- `TF001` delete of a data-holding resource (critical): `aws_db_instance`, `aws_rds_cluster`, `aws_dynamodb_table`, `aws_s3_bucket`, `aws_elasticsearch_domain`, `aws_opensearch_domain`, `google_sql_database_instance`, `google_bigtable_instance`, `azurerm_sql_server`, `azurerm_cosmosdb_account`.
- `TF002` replace of a data-holding resource, that is both delete and create actions (critical).
- `TF003` IAM change: `aws_iam_role`, `aws_iam_policy`, `aws_iam_user`, `aws_iam_role_policy`, `aws_iam_role_policy_attachment`, `google_project_iam_binding`, `google_project_iam_member`, `azurerm_role_assignment`. Critical on delete or replace, medium on create, high otherwise.
- `TF004` network security change: `aws_security_group`, `aws_security_group_rule`, `aws_network_acl`, `google_compute_firewall`, `azurerm_network_security_group`, `azurerm_network_security_rule`. Critical on delete or replace, medium on create, high otherwise.
- `TF005` delete or replace of any other resource (high).

**Helm rules:** `HELM001` release in status `failed`, `pending-install`, `pending-upgrade`, `pending-rollback`, `uninstalling`, or `unknown` (high), `HELM002` superseded release (low), `HELM003` chart lint `[ERROR]` line (high), `HELM004` chart lint `[WARNING]` line (medium), `HELM005` chart lint `[INFO]` line (info).

If `helm lint` fails without printing an `[ERROR]` line (for example, the chart directory has no `Chart.yaml`), the command exits with an error that includes the helm output.

**Kubernetes rules:** `K8S101` missing `metadata.name` (high), `K8S102` deprecated apiVersion such as `extensions/v1beta1` or `policy/v1beta1` (high), `K8S103` Deployment with `replicas: 1` (low), `K8S104` image tag is `:latest` or missing (medium), `K8S105` container without resource requests (medium), `K8S106` container without a liveness probe (low).

Directories are walked recursively for `.yaml` and `.yml` files, skipping `node_modules`, `vendor`, `dist`, `build`, `.git`, and `.angular`. Documents that do not parse as YAML, or that have neither `kind` nor `apiVersion`, are skipped without a warning.

**Plain output:**
```
Infra validation: plan.json
---------------------------
  validated at:        2026-05-16 10:42:07
  result:              FAILED (3 issue(s))

  [1] CRITICAL  TF001  Destructive delete: aws_db_instance.production (RDS database: deletion destroys all data permanently)
  [2] HIGH      TF003  IAM change: aws_iam_role.app_role is a IAM role
  [3] HIGH      TF004  Network security change: aws_security_group.web is a Security group
```

Plain output lists issues by severity, critical first.

**JSON output shape:**
```json
{
  "target": "plan.json",
  "passed": false,
  "issues": [
    {
      "severity": "critical",
      "rule": "TF001",
      "message": "Destructive delete: aws_db_instance.production (RDS database: deletion destroys all data permanently)",
      "resource": "aws_db_instance.production"
    }
  ],
  "validated_at": "2026-05-16T10:42:07Z"
}
```

`target` is `""` for the helm release list. `warnings` lists skipped files and is left out when empty. `issues` is `null` when there are none.

**Exit codes:** `0` validation passed, `1` error, `2` validation FAILED.

---

## `tend cost scan`

```
tend cost scan [flags]
```

Fetch cloud costs and detect anomalies.

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--provider` | string | aws | cloud provider: aws, gcp, azure |
| `--since` | string | 31 days before today | billing period start date (YYYY-MM-DD) |
| `--format` | string | plain | output format: plain, json, table |

**AWS authentication** uses the standard credential chain: `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` environment variables, then `~/.aws/credentials`, then the IAM instance role. The IAM policy requires `ce:GetCostAndUsage` and `ce:GetCostForecast`. Cost Explorer charges $0.01 per API request.

**Period:** costs are fetched per day, grouped by service, from `--since` up to the day before yesterday (the request end date is yesterday, which Cost Explorer treats as exclusive). Services with a total below $0.01 are left out.

**Anomaly detection:** a service is flagged if its cost on the last day of the period is more than 20% above the average of the 7 days before it AND the increase exceeds $5. Both conditions must be met. The period must have more than 7 days.

**Forecast:** a 30-day projected cost is fetched via `GetCostForecast` and displayed if available. Any forecast error, such as too little cost history, leaves the forecast out without a message.

**Plain output:**
```
Cost analysis: AWS
------------------
  period:              2026-04-15 to 2026-05-14
  total cost:          $1206.55 USD
  scanned at:          2026-05-16 10:42:07
  forecast (next 30 days): $1310.00 USD

  TOP SERVICES BY COST (5):

    Amazon Elastic Compute Cloud - Compute         $   612.40  (50.8%)
    Amazon Relational Database Service             $   287.15  (23.8%)
    Amazon Simple Storage Service                  $   156.88  (13.0%)
    AWS Lambda                                     $    88.42  ( 7.3%)
    Amazon CloudFront                              $    61.70  ( 5.1%)

  ANOMALIES (1):

  [1] Amazon Relational Database Service: 34% above 7-day average ($28.71/day vs $21.43/day avg)
```

Plain output shows the top 15 services and then `... and N more services`. Without anomalies it ends with `No cost anomalies detected.`

**JSON output shape:**
```json
{
  "provider": "aws",
  "period": "2026-04-15 to 2026-05-14",
  "total_cost": 1206.55,
  "currency": "USD",
  "records": [
    { "service": "Amazon Elastic Compute Cloud - Compute", "resource": "", "amount": 612.4, "currency": "USD", "period": "2026-04-15 to 2026-05-14" }
  ],
  "anomalies": [
    {
      "service": "Amazon Relational Database Service",
      "message": "34% above 7-day average ($28.71/day vs $21.43/day avg)",
      "expected": 21.43,
      "actual": 28.71,
      "delta_pct": 34.0
    }
  ],
  "by_resource": [
    { "resource": "forecast (next 30 days)", "amount": 1310, "currency": "USD" }
  ],
  "scanned_at": "2026-05-16T10:42:07Z"
}
```

`records` are sorted by amount, highest first, and `anomalies` by `delta_pct`. `resource` in `records` is always `""`. `by_resource` holds only the forecast and is left out when there is none. `anomalies` is `null` when there are none.

**Exit codes:** `0` no anomalies, `1` error, `2` one or more anomalies. `--provider gcp` and `--provider azure` are not implemented and exit `1` with `ERR_NOT_IMPLEMENTED` (see `ERROR_CODES.md`).
