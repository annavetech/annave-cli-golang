# tend

A single static binary that bundles seven engineering tools under one command. No runtime dependencies. No agents. No cloud sign-up.

**License:** MIT

---

## Why this exists

Engineering teams accumulate a pile of one-off scripts: a Bash file that greps the logs, a Python script that pings the health endpoints, a Makefile target that runs `helm list`. They work until they don't: when the machine that wrote them is gone, when the Python version changes, or when the script silently does the wrong thing.

tend replaces that pile with a single binary compiled from typed Go. Every tool is consistent: the same `--format` flag, the same error structure, the same output that pipes cleanly into `jq`.

| Tool | What it does |
|---|---|
| `tend log analyze` | Parse log files, surface spikes, patterns, and clusters |
| `tend health check` | Check HTTP, TCP, and DNS targets concurrently |
| `tend doc search` | Full-text search across local documentation files |
| `tend cleanup scan` | Find idle Kubernetes resources before they cost money |
| `tend security audit` | Scan for secrets, SAST issues, and Kubernetes misconfigs |
| `tend infra validate` | Validate Terraform plans, Helm charts, and K8s manifests |
| `tend cost scan` | Fetch AWS costs and surface anomalies |

---

## Install

**Homebrew (macOS and Linux):**

```bash
brew install annavetech/tap/tend
```

**Go install:**

```bash
go install github.com/annavetech/tend/cmd/tend@latest
```

**Binary download:**

Each [release](https://github.com/annavetech/tend/releases) has a `tar.gz` archive for macOS and Linux on amd64 and arm64, plus `checksums.txt`. Download the archive for your platform (`darwin_arm64` below), verify it, and copy `tend` to a directory in your `$PATH`:

```bash
gh release download --repo annavetech/tend --pattern 'tend_*_darwin_arm64.tar.gz' --pattern checksums.txt
shasum -a 256 -c --ignore-missing checksums.txt
tar -xzf tend_*_darwin_arm64.tar.gz tend
```

**Build from source:**

```bash
git clone https://github.com/annavetech/tend
cd tend
go build -o tend ./cmd/tend
```

Go 1.26 or later is required.

---

## Quick start

```bash
# Analyze a log file
tend log analyze /var/log/app.log

# Check services
tend health check https://api.example.com tcp://db.internal:5432

# Search documentation
tend doc search "authentication" --path ./docs

# Find idle Kubernetes resources
tend cleanup scan --dry-run

# Scan the current directory for secrets
tend security audit .

# Validate a Terraform plan
tend infra validate plan.json

# Fetch AWS costs
tend cost scan
```

Every command supports `--format plain|json|table`. JSON output pipes directly into `jq`:

```bash
tend health check https://api.example.com --format json | jq '.results[].status'
tend cost scan --format json | jq '.records[:5]'
```

---

## Commands

### `tend log analyze`

```
tend log analyze [file] [flags]

Flags:
  --stdin         read from stdin instead of a file
  --format        output format: plain, json, table (default: plain)
  --since         only include entries at or after this time (1h, 30m, 2006-01-02, or RFC3339)
  --level         minimum log level: debug, info, warn, error, critical (warning is accepted as warn)
```

Detects the log format automatically (JSON structured, nginx access, syslog, plain text). Surfaces three types of findings for warn, error, and critical entries: repeated messages (3+ occurrences), error spikes (a minute with at least 5 errors and 3x the average errors per minute), and message clusters (messages that match after numbers, IDs, and addresses are replaced).

```bash
tend log analyze /var/log/nginx/access.log --since 1h --level error
cat app.log | tend log analyze --stdin --format json
```

---

### `tend health check`

```
tend health check [targets...] [flags]

Flags:
  --timeout       per-target timeout (default: 10s, at most 10s)
  --format        output format: plain, json, table (default: plain)
  --chain         check targets in order and stop at the first one that is not up
```

Target formats:

| Format | Check type |
|---|---|
| `https://api.example.com` | HTTP (expects a 2xx response) |
| `tcp://db.internal:5432` | TCP port connectivity |
| `dns://api.example.com` | DNS resolution |
| `db.internal:5432` | TCP (auto-detected from the `:`) |
| `api.example.com` | DNS (auto-detected) |

```bash
tend health check https://api.example.com tcp://redis:6379 dns://internal.corp
tend health check https://api.example.com tcp://db:5432 --chain --timeout 5s
```

---

### `tend doc search`

```
tend doc search [query] [flags]

Flags:
  -p, --path      root directory to search (default: .)
  --format        output format: plain, json, table (default: plain)
  --open          open the top result in $EDITOR (then $VISUAL, then code, nvim, vim, nano, vi)
  --ext           file extensions to include (repeatable: --ext md --ext txt)
```

Searches `.md`, `.mdx`, `.txt`, `.rst`, `.html`, `.htm`, `.yaml`, `.yml`, and `.json` files. Each query word adds to a file's score, so a file matches when any word matches. A word with no whole-word match anywhere matches words that start with it, at half the score.

```bash
tend doc search "rate limiting" --path ./docs
tend doc search timeout --ext md --open
tend doc search "database connection" --format table
```

---

### `tend cleanup scan`

```
tend cleanup scan [flags]

Flags:
  -n, --namespace  limit scan to a specific namespace (default: all)
  --context        kubeconfig context to use
  --kubeconfig     path to kubeconfig file
  --dry-run        label the output as a dry run (default: true); the scan never deletes anything
  --format         output format: plain, json, table (default: plain)
```

Requires a reachable Kubernetes cluster. Uses the standard kubeconfig resolution: the `--kubeconfig` flag, then `$KUBECONFIG`, then `~/.kube/config`.

Finds: completed and failed pods, CrashLoopBackOff pods, pods pending and older than 10 minutes, lost PVCs, pending PVCs older than 1 hour, bound PVCs older than 24 hours that no pod uses, orphaned ConfigMaps (no owner references, older than 30 days), and empty namespaces older than 24 hours.

```bash
tend cleanup scan
tend cleanup scan --namespace production --format table
tend cleanup scan --context staging --dry-run=false
```

---

### `tend security audit`

```
tend security audit [path] [flags]

Flags:
  --type          scan type: secrets, sast, k8s-live, k8s-local (default: secrets)
  --format        output format: plain, json, table (default: plain)
  --kubeconfig    kubeconfig file (k8s-live only)
  --context       kubeconfig context (k8s-live only)
  --namespace     namespace to scan (k8s-live only, default: all namespaces)
```

**Scan types:**

`secrets` scans the file tree for 12 secret patterns (`SECRET001` to `SECRET012`). It does not respect `.gitignore`, so `.env` files are scanned on purpose. Skips binary files, files over 5 MB, and generated directories (`node_modules`, `vendor`, `dist`, `build`, `.git`, `.angular`, `__pycache__`, `.next`, `coverage`, `.cache`).

`sast` runs static analysis pattern matching. Go: 10 rules, `GO001` to `GO010` (command injection, SQL injection, path traversal, SSRF, weak random, weak crypto, plaintext HTTP server, unbounded read, unsafe import, hardcoded credentials). TypeScript and JavaScript: 5 rules, `TS001` to `TS005` (innerHTML XSS, dangerouslySetInnerHTML, eval, document.write, sensitive localStorage).

`k8s-live` checks running pods via client-go with 7 rules, `K8S001` to `K8S007`: hostNetwork, hostPID, hostPath volumes, privileged containers, containers that may run as root, missing resource limits, and missing readiness probes.

`k8s-local` validates local YAML manifests. Same rules as `k8s-live` plus `K8S008`, unpinned image tags.

```bash
tend security audit .
tend security audit ./src --type sast
tend security audit --type k8s-live --context production
tend security audit ./k8s --type k8s-local --format table
```

---

### `tend infra validate`

```
tend infra validate [target] [flags]

Flags:
  --type          validate type: terraform, helm, k8s (default: auto-detect)
  --format        output format: plain, json, table (default: plain)
```

Target auto-detection:

| Target | Detected as |
|---|---|
| `*.json`, `*.tfplan` | terraform plan analysis |
| directory containing `Chart.yaml` | helm chart lint |
| any other file or directory | Kubernetes manifest validation |
| no target, or `.` | helm release status (`helm list -A --all`) |

A `.tfplan` file needs `terraform` in `$PATH`; a `.json` plan is read directly. Helm checks need `helm` in `$PATH`. The command returns a clear error if a needed CLI is not found. The validation fails when any issue is medium or above.

```bash
tend infra validate plan.json
tend infra validate ./charts/myapp
tend infra validate ./k8s --format table
tend infra validate --type helm
```

---

### `tend cost scan`

```
tend cost scan [flags]

Flags:
  --provider      cloud provider: aws, gcp, azure (default: aws)
  --since         billing period start date (YYYY-MM-DD, default: 31 days before today)
  --format        output format: plain, json, table (default: plain)
```

**AWS authentication** uses the standard credential chain:
1. `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` environment variables
2. `~/.aws/credentials` file
3. IAM instance role (EC2, ECS, Lambda)

Note: AWS Cost Explorer charges $0.01 per API request.

GCP and Azure are not yet implemented; the command prints an `ERR_NOT_IMPLEMENTED` error and exits 1.

```bash
tend cost scan
tend cost scan --since 2026-04-01
tend cost scan --provider aws --format table
```

---

## Output formats

Every command supports `--format` with three values:

- `plain`: human-readable terminal output (default)
- `json`: structured JSON, suitable for piping into `jq` or other tools
- `table`: tabular output for wide terminals

JSON keys are snake_case and match the domain model of each command. Durations are given in milliseconds (`latency_ms`, `timeout_ms`). See `docs/COMMANDS.md` for the exact JSON shapes per command.

Exit codes are the same for every command: `0` nothing to report, `1` error, `2` problems found (findings, down targets, idle resources, anomalies, or a failed validation).

---

## Configuration

Limits are embedded in the binary from `internal/shared/config/limits.yaml`. To change a limit, edit the YAML and rebuild.

| File | Controls |
|---|---|
| `internal/shared/config/limits.yaml` | Log line cap, health timeout and concurrency, doc search size, depth, and result limits, cleanup timeout and result cap |

See `docs/CONFIGURATION.md` for every key and its default value.

---

## Architecture

tend uses **hexagonal architecture** (ports and adapters). Each module has a domain model, a port interface, and one or more adapter implementations. The CLI layer is thin. It parses flags, calls the port, and formats the output. Nothing in the domain core knows about Cobra or terminal formatting.

See `docs/ARCHITECTURE.md` for the full package map and instructions for adding a new module.

---

## Third-party attributions

See [NOTICE](NOTICE) for the full list of open-source components included in this project.
