<!--
title:       tend: Use Cases
description: Seven practical scenarios with real shell examples and notes on edge cases.
             No marketing language, just what works and what to watch for.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Use Cases

Seven practical scenarios, one per module. Each includes working shell examples and notes on edge cases and limitations.

---

## 1. Investigating a production incident via logs

**Scenario:** An alert fired. You have a log file from the affected pod. You want to know what changed in the last hour.

```bash
tend log analyze /var/log/app/api.log --since 1h --level error
```

For log streaming from kubectl:

```bash
kubectl logs -n production deploy/api --since=1h > /tmp/api.log
tend log analyze /tmp/api.log --since 1h --level error
```

Pipe the end of a log from another command. The analysis runs when the input ends, so pass a finite stream, not `tail -f`:

```bash
tail -n 50000 /var/log/app.log | tend log analyze --stdin --level error
```

For machine-readable output to feed into another tool:

```bash
tend log analyze /var/log/app.log --format json | jq '.findings[] | select(.severity == "critical")'
```

**What to watch:**
- The analyzer detects the format from the first non-blank line and applies it to every line. If a log file mixes formats (JSON lines then plain text after a restart), lines that do not match keep the whole line as the message, with no timestamp. `parsed_lines` counts every non-blank line.
- The `--since` filter applies to the timestamp parsed from each log entry, not to the file modification time. Entries without a parseable timestamp are always included.
- Spike detection needs at least 5 error entries in at least 2 different minutes. A minute is a spike when it has at least 5 errors and at least 3x the average of the minutes that have errors. A log that covers one minute produces no spike findings.

---

## 2. Pre-deploy health check across all services

**Scenario:** Before deploying a new version, verify that all upstream dependencies are reachable.

```bash
tend health check \
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
tend health check \
  tcp://postgres:5432 \
  tcp://redis:6379 \
  https://api.internal/health \
  --chain
```

In `--chain` mode, the command stops at the first failure. If postgres is down, redis and the API are not checked, on the assumption that they depend on the database.

Fail a CI step if any target is down. The command exits `2` when a target is down and `1` on an error:

```bash
tend health check https://api.example.com
```

**What to watch:**
- HTTP checks follow up to 4 redirects. A `301` to HTTPS from an `http://` target counts as up if the final response is 2xx.
- TCP checks only verify that the port accepts a connection. They do not send any protocol bytes or verify TLS.
- DNS checks call the system resolver. Split-horizon DNS (internal vs external) means the result depends on where `tend` is running. Running it inside the cluster gives different results than running it from a developer laptop.

---

## 3. Searching across a large documentation repository

**Scenario:** A monorepo has hundreds of Markdown files spread across multiple packages. You need to find where authentication timeout is configured.

```bash
tend doc search "authentication timeout" --path ./docs
```

Open the top result immediately in your editor:

```bash
tend doc search "rate limit" --path ./docs --open
```

Restrict to a specific file type:

```bash
tend doc search "database connection" --ext md --ext txt --path .
```

Search in the current directory (walks recursively):

```bash
tend doc search "circuit breaker" --format table
```

**What to watch:**
- The index is built fresh on each run. There is no persistent index file. For a directory with many thousands of files this adds latency. Directories more than 10 levels below the search root are skipped; change `doc.max_depth` in `limits.yaml` if needed.
- Files larger than 10 MB are skipped. This is configurable in `limits.yaml`.
- `--open` uses `$EDITOR`, then `$VISUAL`. Without either, it uses the first of `code`, `nvim`, `vim`, `nano` found on `PATH`, and `vi` otherwise. Line-awareness is only available for `nvim`, `vim`, `code`, and `nano`.
- The index keeps `-` and `_` inside words, so `rate-limit` and `rate_limit` are single tokens. A search for `rate` matches them by prefix only when no file has `rate` as a whole word. Words shorter than 3 characters are not indexed.

---

## 4. Weekly Kubernetes hygiene scan

**Scenario:** Run a weekly scan to find resources that are consuming quota but doing nothing.

```bash
tend cleanup scan --format table
```

Scope to a single namespace:

```bash
tend cleanup scan --namespace staging --format table
```

Export results for a report:

```bash
tend cleanup scan --format json | jq '.resources | group_by(.resource.kind) | map({kind: .[0].resource.kind, count: length})'
```

Use a specific kubeconfig context without changing the current context:

```bash
tend cleanup scan --context production-eu --kubeconfig ~/.kube/prod.yaml
```

**What to watch:**
- The scan is read-only. It never deletes anything, with or without `--dry-run=false`.
- ConfigMap orphan detection only checks `ownerReferences`. A ConfigMap referenced only by label selector (not by owner reference) will appear as orphaned. Verify before acting.
- Namespace emptiness is checked using `Limit: 1` list queries for pods, deployments, StatefulSets, and services. Other resource types (CronJobs, Jobs, DaemonSets) are not checked. A namespace with only CronJobs will appear empty. Namespaces created in the last 24 hours and `default` are not flagged.
- Large clusters with many resources may hit the `max_resources: 1000` limit in `limits.yaml`. Increase it if needed, but be aware that the scan timeout also applies.

---

## 5. Scanning a new repository for secrets before the first push

**Scenario:** You cloned a codebase from a vendor. Before pushing it to your own GitHub, scan it for hardcoded credentials.

```bash
tend security audit /path/to/vendor-repo
```

Scan only the application source, not the test fixtures:

```bash
tend security audit ./src --type secrets --format table
```

Run SAST on the Go backend:

```bash
tend security audit ./backend --type sast
```

Export for a security review report:

```bash
tend security audit . --type secrets --format json | jq '.findings | map(select(.severity == "high" or .severity == "critical"))'
```

**What to watch:**
- The scanner does NOT respect `.gitignore`. This is intentional: `.env` files are in `.gitignore` precisely because they contain secrets, so they must be scanned.
- Binary files are skipped (detected by a zero byte in the first 512 bytes). Files over 5 MB are skipped too.
- SAST uses regex pattern matching, not an AST. This means false positives are possible: a comment containing `http.ListenAndServe(` will match rule `GO007`. Review findings before acting.
- The SAST scanner will flag its own rule definition strings if you run it against this repository. This is a known limitation of regex SAST without AST support.
- Secret values in findings are redacted: the first 12 characters are shown, followed by up to 8 asterisks and `...`. This is enough to identify the secret without logging the full value.

---

## 6. Validating a Terraform plan before apply

**Scenario:** A pull request changes infrastructure. You want to surface destructive changes before anyone approves it.

```bash
# Generate the plan JSON
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json

# Validate it
tend infra validate plan.json
```

Or if you already have the JSON plan:

```bash
tend infra validate plan.json --format table
```

In CI, where the command exits `2` when the validation fails (any medium, high, or critical issue):

```bash
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
tend infra validate plan.json
```

Lint a Helm chart before packaging:

```bash
tend infra validate ./charts/myapp
```

Check all deployed Helm releases across all namespaces:

```bash
tend infra validate --type helm --format table
```

**What to watch:**
- `terraform` must be in `$PATH`. If it is not found, the command returns a clear error. The validator also accepts a raw `terraform show -json` JSON file. It does not need to call the binary if you pass the file directly.
- `helm` must be in `$PATH` for release listing and chart linting. The validator checks for the binary at runtime and returns a clear error if it is missing.
- Terraform validation reads `resource_changes` from the plan JSON. It does not evaluate the actual change. It classifies by resource type and action. A `replace` on an EC2 instance is flagged as destructive even if the instance has no data.
- Kubernetes YAML validation walks directories recursively. Documents that do not parse as valid YAML are skipped without a warning. A file that cannot be read is named in a `warning:` line.

---

## 7. Monthly AWS cost review

**Scenario:** Review which services drove the cost increase last month and flag any anomalies.

```bash
tend cost scan --since 2026-04-01
```

Compare two periods by running twice:

```bash
tend cost scan --since 2026-04-01 --format json > april.json
tend cost scan --since 2026-05-01 --format json > may.json
jq -s '[.[0].records, .[1].records | .[] | {service: .service, amount: .amount}] | group_by(.service)' april.json may.json
```

Export for a spreadsheet:

```bash
tend cost scan --format json | jq -r '.records[] | [.service, .amount] | @csv'
```

**What to watch:**
- The scan sets the request end date to yesterday. Cost Explorer treats the end date as exclusive, so the last day in the report is the day before yesterday.
- If `GetCostForecast` returns an error, for example because the account has too little cost history, the forecast is omitted without a message, and the cost data is still returned.
- The Cost Explorer API is global. It does not use the AWS region set in `~/.aws/config`. If you have regional credential profiles, the default profile is used unless `AWS_PROFILE` is set.
- Cost Explorer charges $0.01 per API request. Each `tend cost scan` run makes 2 requests (cost data + forecast). Running it in a tight loop will accumulate charges.
- GCP and Azure are not yet implemented. The command prints an error and exits `1`.
