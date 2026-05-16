<!--
title:       ANNÁVE CLI — Use Cases
description: Seven practical scenarios with real shell examples and notes on edge cases.
             No marketing language — just what works and what to watch for.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Use Cases

Seven practical scenarios — one per module. Each includes working shell examples and notes on edge cases and limitations.

---

## 1. Investigating a production incident via logs

**Scenario:** An alert fired. You have a log file from the affected pod. You want to know what changed in the last hour.

```bash
annave log analyze /var/log/app/api.log --since 1h --level error
```

For log streaming from kubectl:

```bash
kubectl logs -n production deploy/api --since=1h > /tmp/api.log
annave log analyze /tmp/api.log --since 1h --level error
```

Pipe from a running process:

```bash
tail -f /var/log/app.log | annave log analyze --stdin --level error
```

For machine-readable output to feed into another tool:

```bash
annave log analyze /var/log/app.log --format json | jq '.findings[] | select(.severity == "critical")'
```

**What to watch:**
- The analyzer detects format automatically. If a log file mixes formats (JSON lines then plain text after a restart), lines that do not match the detected format are counted in `total_lines` but not in `parsed_lines`.
- The `--since` filter applies to the timestamp parsed from each log entry, not to the file modification time. Entries without a parseable timestamp are always included.
- Spike detection requires at least 10 minutes of log data to compute a meaningful rolling average. Short log files may produce no spike findings even if one minute has many errors.

---

## 2. Pre-deploy health check across all services

**Scenario:** Before deploying a new version, verify that all upstream dependencies are reachable.

```bash
annave health check \
  https://api.stripe.com/v1 \
  https://api.sendgrid.com \
  tcp://db.internal:5432 \
  tcp://redis.internal:6379 \
  dns://auth.internal \
  --timeout 5s \
  --format table
```

For a dependency chain where each service requires the previous:

```bash
annave health check \
  tcp://postgres:5432 \
  tcp://redis:6379 \
  https://api.internal/health \
  --chain
```

In `--chain` mode, the command stops at the first failure. If postgres is down, redis and the API are not checked — the assumption is they depend on the database.

Exit in CI if any target is down:

```bash
annave health check https://api.example.com --format json \
  | jq -e '.down == 0' > /dev/null \
  || { echo "health check failed"; exit 1; }
```

**What to watch:**
- HTTP checks follow up to 5 redirects. A `301` to HTTPS from an `http://` target counts as up if the final response is 2xx.
- TCP checks only verify that the port accepts a connection — they do not send any protocol bytes or verify TLS.
- DNS checks call the system resolver. Split-horizon DNS (internal vs external) means the result depends on where `annave` is running. Running it inside the cluster gives different results than running it from a developer laptop.

---

## 3. Searching across a large documentation repository

**Scenario:** A monorepo has hundreds of Markdown files spread across multiple packages. You need to find where authentication timeout is configured.

```bash
annave doc search "authentication timeout" --path ./docs
```

Open the top result immediately in your editor:

```bash
annave doc search "rate limit" --path ./docs --open
```

Restrict to a specific file type:

```bash
annave doc search "database connection" --ext md --ext txt --path .
```

Search in the current directory (walks recursively):

```bash
annave doc search "circuit breaker" --format table
```

**What to watch:**
- The index is built fresh on each run — there is no persistent index file. For a directory with many thousands of files this adds latency. The default max depth is 10 levels; override it in `limits.yaml` if needed.
- Files larger than 10 MB are skipped. This is configurable in `limits.yaml`.
- `--open` without a `$EDITOR` set falls back to the OS default application for the file type (macOS `open`, Linux `xdg-open`). Line-awareness is only available for `nvim`, `vim`, `code`, and `nano`.
- The inverted index tokenises on word boundaries. Hyphenated words (`rate-limit`) are split into tokens (`rate`, `limit`). Search for `rate limit` to match both forms.

---

## 4. Weekly Kubernetes hygiene scan

**Scenario:** Run a weekly scan to find resources that are consuming quota but doing nothing.

```bash
annave cleanup scan --format table
```

Scope to a single namespace:

```bash
annave cleanup scan --namespace staging --format table
```

Export results for a report:

```bash
annave cleanup scan --format json | jq '.resources | group_by(.resource.kind) | map({kind: .[0].resource.kind, count: length})'
```

Use a specific kubeconfig context without changing the current context:

```bash
annave cleanup scan --context production-eu --kubeconfig ~/.kube/prod.yaml
```

**What to watch:**
- The scan is read-only and dry-run by default. It will never delete anything.
- ConfigMap orphan detection only checks `ownerReferences`. A ConfigMap referenced only by label selector (not by owner reference) will appear as orphaned. Verify before acting.
- Namespace emptiness is checked using `Limit: 1` list queries for pods, deployments, StatefulSets, and services. Other resource types (CronJobs, Jobs, DaemonSets) are not checked. A namespace with only CronJobs will appear empty.
- Large clusters with many resources may hit the `max_resources: 1000` limit in `limits.yaml`. Increase it if needed, but be aware that the scan timeout also applies.

---

## 5. Scanning a new repository for secrets before the first push

**Scenario:** You cloned a codebase from a vendor. Before pushing it to your own GitHub, scan it for hardcoded credentials.

```bash
annave security audit /path/to/vendor-repo
```

Scan only the application source, not the test fixtures:

```bash
annave security audit ./src --type secrets --format table
```

Run SAST on the Go backend:

```bash
annave security audit ./backend --type sast
```

Export for a security review report:

```bash
annave security audit . --type secrets --format json | jq '.findings | map(select(.severity == "high" or .severity == "critical"))'
```

**What to watch:**
- The scanner does NOT respect `.gitignore`. This is intentional — `.env` files are in `.gitignore` precisely because they contain secrets, so they must be scanned.
- Binary files are skipped (detected by null bytes in the first 512 bytes).
- SAST uses regex pattern matching, not an AST. This means false positives are possible — a comment containing `http.ListenAndServe` will match rule `GO008`. Review findings before acting.
- The SAST scanner will flag its own rule definition strings if you run it against this repository. This is a known limitation of regex SAST without AST support.
- Secret values in findings are redacted: the first 12 characters are shown and the rest is masked with asterisks. This is enough to identify the secret without logging the full value.

---

## 6. Validating a Terraform plan before apply

**Scenario:** A pull request changes infrastructure. You want to surface destructive changes before anyone approves it.

```bash
# Generate the plan JSON
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json

# Validate it
annave infra validate plan.json
```

Or if you already have the JSON plan:

```bash
annave infra validate plan.json --format table
```

In CI:

```bash
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
annave infra validate plan.json --format json \
  | jq -e '.passed == true' > /dev/null \
  || { echo "infra validation failed"; exit 1; }
```

Lint a Helm chart before packaging:

```bash
annave infra validate ./charts/myapp
```

Check all deployed Helm releases across all namespaces:

```bash
annave infra validate --type helm --format table
```

**What to watch:**
- `terraform` must be in `$PATH`. If it is not found, the command returns a clear error. The validator also accepts a raw `terraform show -json` JSON file — it does not need to call the binary if you pass the file directly.
- `helm` must be in `$PATH` for release listing and chart linting. The validator checks for the binary at runtime and returns a clear error if it is missing.
- Terraform validation reads `resource_changes` from the plan JSON. It does not evaluate the actual change — it classifies by resource type and action. A `replace` on an EC2 instance is flagged as destructive even if the instance has no data.
- Kubernetes YAML validation walks directories recursively. Files that do not parse as valid YAML are skipped with a warning, not an error.

---

## 7. Monthly AWS cost review

**Scenario:** Review which services drove the cost increase last month and flag any anomalies.

```bash
annave cost scan --since 2026-04-01
```

Compare two periods by running twice:

```bash
annave cost scan --since 2026-04-01 --format json > april.json
annave cost scan --since 2026-05-01 --format json > may.json
jq -s '[.[0].records, .[1].records | .[] | {service: .service, amount: .amount}] | group_by(.service)' april.json may.json
```

Export for a spreadsheet:

```bash
annave cost scan --format json | jq -r '.records[] | [.service, .amount] | @csv'
```

**What to watch:**
- AWS Cost Explorer data has a 24-hour delay. The scan always sets the end date to yesterday. You cannot fetch today's costs.
- For accounts with less than 14 days of data, `GetCostForecast` returns an error. The forecast is silently omitted — the cost data is still returned.
- The Cost Explorer API is global. It does not use the AWS region set in `~/.aws/config`. If you have regional credential profiles, the default profile is used unless `AWS_PROFILE` is set.
- Cost Explorer charges $0.01 per API request. Each `annave cost scan` run makes 2 requests (cost data + forecast). Running it in a tight loop will accumulate charges.
- GCP and Azure display a notice that they are not yet implemented. The exit code is still `0`.
