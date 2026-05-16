# Changelog

All notable changes to ANNÁVE CLI are documented here.
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.1.0] — 2026-05-16

### Added

- **`annave log analyze`** — log file analysis for JSON, nginx, syslog, and plain formats. Detects repeated error patterns, time-based spikes, and message clusters. Supports file path, `--stdin`, `--since`, and `--level` filtering.

- **`annave health check`** — concurrent HTTP, TCP, and DNS health checks with latency measurement. Auto-detects target type from URL scheme or format. `--chain` mode stops at the first failure for dependency chains.

- **`annave doc search`** — full-text search across local documentation files (`.md`, `.txt`, `.rst`, `.html`, `.yaml`, `.json`). Inverted index with exact and prefix matching. `--open` opens the top result in `$EDITOR` with line-awareness for vim, nvim, and VS Code.

- **`annave cleanup scan`** — Kubernetes idle resource detection via client-go. Finds completed and failed pods, CrashLoopBackOff pods, pending-too-long pods, lost PVCs, stale bound PVCs, orphaned ConfigMaps, and empty namespaces. Dry-run by default.

- **`annave security audit`** — security scanning with four modes:
  - `secrets` — 12 regex patterns covering AWS keys, GCP service accounts, RSA/EC private keys, JWT secrets, API tokens, database URLs with passwords, GitHub tokens, Slack tokens. Skips binary files and generated directories. Does not respect `.gitignore` — scans `.env` files deliberately.
  - `sast` — static analysis for Go (10 rules: command injection, SQL injection, path traversal, SSRF, weak random, weak crypto, plaintext server, unbounded read, unsafe import, hardcoded credentials) and TypeScript (5 rules: innerHTML XSS, dangerouslySetInnerHTML, eval, document.write, sensitive localStorage).
  - `k8s-live` — checks running cluster workloads via kubeconfig for 7 security misconfigurations.
  - `k8s-local` — validates local Kubernetes YAML manifests against the same rule set plus unpinned image tags.

- **`annave infra validate`** — infrastructure definition validation with three modes:
  - `terraform` — parses `terraform show -json` output or direct plan JSON. Flags destructive changes, IAM modifications, network exposure, and data deletion risk.
  - `helm` — lists deployed releases via `helm list -A -o json` and lints charts via `helm lint`. Detects failed and pending releases.
  - `k8s` — validates Kubernetes YAML manifests for deprecated API versions, missing metadata, latest image tags, single-replica deployments without PodDisruptionBudget, missing resource limits, and missing liveness probes.

- **`annave cost scan`** — AWS cloud cost analysis via Cost Explorer API. Daily granularity grouped by service. 7-day rolling anomaly detection (flags services more than 20% above average and more than $5 absolute increase). Optional 30-day cost forecast. GCP and Azure are stubs with a friendly notice.

- Shared `--format plain|json|table` flag on every command.
- Shared `AnnaveError` structured error type with code, stage, and message fields.
- Embedded YAML configuration for limits and messages (`internal/shared/config/`).
- Apache 2.0 license, SPDX headers on all source files.
