# ANNÁVE CLI

A single static binary that bundles seven engineering tools under one command. No runtime dependencies. No agents. No cloud sign-up.

**Author:** Anna Veretennykova · [www.annave.tech](https://www.annave.tech)
**License:** Apache 2.0

---

## Why this exists

Engineering teams accumulate a pile of one-off scripts: a Bash file that greps the logs, a Python script that pings the health endpoints, a Makefile target that runs `helm list`. They work until they don't — when the machine that wrote them is gone, or when the Python version changes, or when the script silently does the wrong thing.

ANNAVE CLI replaces that pile with a single binary compiled from typed Go. Every tool is consistent: the same `--format` flag, the same error structure, the same output that pipes cleanly into `jq`.

| Tool | What it does |
|---|---|
| `annave log analyze` | Parse log files, surface spikes, patterns, and clusters |
| `annave health check` | Check HTTP, TCP, and DNS targets concurrently |
| `annave doc search` | Full-text search across local documentation files |
| `annave cleanup scan` | Find idle Kubernetes resources before they cost money |
| `annave security audit` | Scan for secrets, SAST issues, and Kubernetes misconfigs |
| `annave infra validate` | Validate Terraform plans, Helm charts, and K8s manifests |
| `annave cost scan` | Fetch AWS costs and surface anomalies |

---

## Install

**Homebrew (macOS and Linux):**

```bash
brew install annavetech/annave/annave
```

**Go install:**

```bash
go install annave.tech/cli/cmd/annave@latest
```

**Binary download:**

Download a pre-built static binary from the [releases page](https://github.com/annavetech/annave-cli-golang/releases). No installation required — copy to a directory in your `$PATH`.

**Build from source:**

```bash
git clone https://github.com/annavetech/annave-cli-golang
cd annave-cli-golang
go build -o annave ./cmd/annave
```

Go 1.24 or later is required.

---

## Quick start

```bash
# Analyze a log file
annave log analyze /var/log/app.log

# Check services
annave health check https://api.example.com tcp://db.internal:5432

# Search documentation
annave doc search "authentication" --path ./docs

# Find idle Kubernetes resources
annave cleanup scan --dry-run

# Scan the current directory for secrets
annave security audit .

# Validate a Terraform plan
annave infra validate plan.json

# Fetch AWS costs
annave cost scan
```

Every command supports `--format plain|json|table`. JSON output pipes directly into `jq`:

```bash
annave health check https://api.example.com --format json | jq '.results[].status'
annave cost scan --format json | jq '.records[:5]'
```

---

## Commands

### `annave log analyze`

```
annave log analyze [file] [flags]

Flags:
  --stdin         read from stdin instead of a file
  --format        output format: plain, json, table (default: plain)
  --since         only include entries after this time (1h, 30m, 2006-01-02, or RFC3339)
  --level         minimum log level: debug, info, warn, error
```

Detects the log format automatically (JSON structured, nginx access, syslog, plain text). Surfaces three types of findings: repeated error patterns (3+ occurrences), time-based spikes (3× the rolling average), and message clusters (normalised duplicates grouped together).

```bash
annave log analyze /var/log/nginx/access.log --since 1h --level error
cat app.log | annave log analyze --stdin --format json
```

---

### `annave health check`

```
annave health check [targets...] [flags]

Flags:
  --timeout       per-target timeout (default: 10s)
  --format        output format: plain, json, table (default: plain)
  --chain         stop at first failure (dependency chain mode)
```

Target formats:

| Format | Check type |
|---|---|
| `https://api.example.com` | HTTP — expects 2xx response |
| `tcp://db.internal:5432` | TCP port connectivity |
| `dns://api.example.com` | DNS resolution |
| `db.internal:5432` | TCP (auto-detected) |
| `api.example.com` | DNS (auto-detected) |

```bash
annave health check https://api.example.com tcp://redis:6379 dns://internal.corp
annave health check https://api.example.com tcp://db:5432 --chain --timeout 5s
```

---

### `annave doc search`

```
annave doc search [query] [flags]

Flags:
  -p, --path      root directory to search (default: .)
  --format        output format: plain, json, table (default: plain)
  --open          open the top result in $EDITOR
  --ext           file extensions to include (repeatable: --ext md --ext txt)
```

Searches `.md`, `.txt`, `.rst`, `.html`, `.yaml`, and `.json` files. Multiple words are treated as AND terms. Prefix matching is supported.

```bash
annave doc search "rate limiting" --path ./docs
annave doc search timeout --ext md --open
annave doc search "database connection" --format table
```

---

### `annave cleanup scan`

```
annave cleanup scan [flags]

Flags:
  -n, --namespace  limit scan to a specific namespace (default: all)
  --context        kubeconfig context to use
  --kubeconfig     path to kubeconfig file
  --dry-run        preview only, make no changes (default: true)
  --format         output format: plain, json, table (default: plain)
```

Requires a reachable Kubernetes cluster. Uses the standard kubeconfig resolution: `--kubeconfig` flag → `$KUBECONFIG` → `~/.kube/config`.

Finds: completed and failed pods, CrashLoopBackOff pods, pods pending longer than 10 minutes, lost PVCs, PVCs pending longer than 1 hour, bound PVCs unmounted for over 24 hours, orphaned ConfigMaps (no owner references, older than 30 days), and empty namespaces.

```bash
annave cleanup scan
annave cleanup scan --namespace production --format table
annave cleanup scan --context staging --dry-run=false
```

---

### `annave security audit`

```
annave security audit [path] [flags]

Flags:
  --type          scan type: secrets, sast, k8s-live, k8s-local (default: secrets)
  --format        output format: plain, json, table (default: plain)
  --kubeconfig    kubeconfig file (k8s-live only)
  --context       kubeconfig context (k8s-live only)
```

**Scan types:**

`secrets` — scans the file tree for 12 secret patterns. Does not respect `.gitignore` — `.env` files are scanned deliberately. Skips binary files and generated directories (`node_modules`, `vendor`, `dist`, `.git`, `.angular`).

`sast` — static analysis pattern matching. Go: 10 rules (command injection, SQL injection, path traversal, SSRF, weak random, weak crypto, plaintext HTTP server, unbounded read, unsafe import, hardcoded credentials). TypeScript: 5 rules (innerHTML XSS, dangerouslySetInnerHTML, eval, document.write, sensitive localStorage).

`k8s-live` — checks running workloads via client-go. Flags containers running as root, missing resource limits, privileged containers, hostPath mounts, missing readiness probes, hostNetwork, and hostPID.

`k8s-local` — validates local YAML manifests. Same rules as `k8s-live` plus unpinned image tags.

```bash
annave security audit .
annave security audit ./src --type sast
annave security audit --type k8s-live --context production
annave security audit ./k8s --type k8s-local --format table
```

---

### `annave infra validate`

```
annave infra validate [target] [flags]

Flags:
  --type          validate type: terraform, helm, k8s (default: auto-detect)
  --format        output format: plain, json, table (default: plain)
```

Target auto-detection:

| Target | Detected as |
|---|---|
| `*.json`, `*.tfplan` | terraform plan analysis |
| directory containing `Chart.yaml` | helm chart lint |
| `*.yaml`, `*.yml`, directory | Kubernetes manifest validation |
| no target | helm release status (`helm list -A`) |

`terraform` and `helm` require their respective CLIs in `$PATH`. The command returns a clear error if they are not found.

```bash
annave infra validate plan.json
annave infra validate ./charts/myapp
annave infra validate ./k8s --format table
annave infra validate --type helm
```

---

### `annave cost scan`

```
annave cost scan [flags]

Flags:
  --provider      cloud provider: aws, gcp, azure (default: aws)
  --since         billing period start date (YYYY-MM-DD, default: 30 days ago)
  --format        output format: plain, json, table (default: plain)
```

**AWS authentication** uses the standard credential chain:
1. `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` environment variables
2. `~/.aws/credentials` file
3. IAM instance role (EC2, ECS, Lambda)

Note: AWS Cost Explorer charges $0.01 per API request.

GCP and Azure display a notice that they are not yet implemented.

```bash
annave cost scan
annave cost scan --since 2026-04-01
annave cost scan --provider aws --format table
```

---

## Output formats

Every command supports `--format` with three values:

- `plain` — human-readable terminal output (default)
- `json` — structured JSON, suitable for piping into `jq` or other tools
- `table` — tabular output for wide terminals

JSON output follows the same structure across all commands — the top-level keys match the domain model. See `docs/COMMANDS.md` for the exact JSON shapes per command.

---

## Configuration

Limits and messages are embedded in the binary from YAML files in `internal/shared/config/`. To change a limit, edit the YAML and rebuild.

| File | Controls |
|---|---|
| `internal/shared/config/limits.yaml` | File size limits, timeouts, max results per module |
| `internal/shared/config/messages.yaml` | Error code to message mapping |

See `docs/CONFIGURATION.md` for every key and its default value.

---

## Architecture

ANNAVE CLI uses **hexagonal architecture** (ports and adapters). Each module has a domain model, a port interface, and one or more adapter implementations. The CLI layer is thin — it parses flags, calls the port, and formats the output. Nothing in the domain core knows about Cobra or terminal formatting.

See `docs/ARCHITECTURE.md` for the full package map and instructions for adding a new module.

---

## Third-party attributions

See [NOTICE](NOTICE) for the full list of open-source components included in this project.
# annave-cli-golang
