<!--
title:       ANNÁVE CLI — Security Notes
description: What the security audit checks, known limitations and false positives,
             how authentication works for live checks, and responsible disclosure.
author:      Anna Veretennykova
website:     www.annave.tech
version:     0.1.0
created:     2026-05-16
updated:     2026-05-16
-->

# Security Notes

---

## What `annave security audit` checks

The audit command has four scan types. Each targets a different attack surface.

### Secret scanning (`--type secrets`)

Walks the file tree and matches 12 regex patterns against every non-binary file. The patterns are designed to find credentials that were accidentally committed, not credentials that are correctly stored in a secrets manager.

**What it scans:**
- All files in the target directory, recursively
- Files in `.env`, `.env.backup`, `.env.local`, and similar names
- YAML and JSON configuration files
- Shell scripts and Makefiles

**What it deliberately skips:**
- Binary files (detected by null bytes in the first 512 bytes of the file)
- Generated directories: `node_modules`, `vendor`, `dist`, `.git`, `.angular`

**Critical design choice — `.gitignore` is ignored.** The scanner does not read or respect `.gitignore`. The reason: `.env` files are in `.gitignore` precisely because they contain secrets. A secret scanner that skips `.gitignore` entries would miss the most common source of leaked credentials. If you want to scan only committed files, run `git diff HEAD -- . | annave log analyze --stdin` or use a git-aware tool like `truffleHog` in addition to this scanner.

**Secret values in output are redacted.** The first 12 characters of a matched secret are shown; the remainder is replaced with asterisks. This is enough to identify which credential was found without logging the full value.

### SAST (`--type sast`)

Line-by-line regex pattern matching for security-relevant code patterns. Not an AST-based analysis — this is a practical trade-off for a developer CLI.

**Known limitation: false positives on rule definitions.** If you run `annave security audit --type sast .` against this repository, the SAST scanner will flag its own rule definition strings (the regex patterns contain the strings they match). This is expected and harmless. Run SAST against your project's source, not against this tool.

**Go rules (10):** Command injection (`exec.Command` + string concat), SQL injection (`db.Query`/`db.Exec` + string concat), path traversal (`os.Open` without `filepath.Clean`), SSRF (`http.Get`/`http.Post` with variable URL), weak random (`math/rand`), weak crypto (`crypto/md5`, `crypto/sha1`), plaintext HTTP server (`http.ListenAndServe` without TLS), unbounded read (`io.ReadAll` without `io.LimitReader`), unsafe memory (`import "unsafe"`), hardcoded credential (assignment to `password`, `secret`, or `token` variable with a string literal).

**TypeScript rules (5):** innerHTML XSS, `dangerouslySetInnerHTML` (React), `eval()`, `document.write()`, sensitive data in `localStorage`.

**What it does not check:** Call graphs, data flow, taint analysis, or cross-file analysis. A `db.Query` call that receives its argument from a function parameter will not be flagged unless the pattern matches at the call site. For deeper analysis, use a dedicated SAST tool (Semgrep, CodeQL).

### Kubernetes live checks (`--type k8s-live`)

Reads live workload definitions from the cluster API. Does not modify any resource. Requires `list` and `get` permissions on pods in the target namespace.

**Checked rules:** Container running as root or without `runAsNonRoot: true` (K8S001), no CPU limit (K8S002), no memory limit (K8S003), privileged container (`privileged: true`) (K8S004), `hostPath` volume mount (K8S005), no readiness probe on a Running pod (K8S006), `hostNetwork: true` or `hostPID: true` (K8S007).

### Kubernetes local manifest checks (`--type k8s-local`)

Parses YAML files from disk. Does not require a cluster connection. Handles multi-document YAML files (documents separated by `---`).

Same rules as `k8s-live` plus K8S008: image tag is `:latest` or missing (non-deterministic deployments).

---

## Authentication for live checks

### `annave security audit --type k8s-live`

Uses the standard kubeconfig resolution:

1. `--kubeconfig` flag
2. `$KUBECONFIG` environment variable
3. `~/.kube/config`

The `--context` flag selects a specific kubeconfig context without modifying the default. Without `--context`, the current context is used.

The scanner uses the same client-go credential loading as `kubectl`. Supported credential types: certificate, token, exec plugin (for cloud provider auth: `gke-gcloud-auth-plugin`, `aws eks get-token`, etc.).

Minimum required RBAC permissions:
```yaml
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["list", "get"]
```

### `annave cost scan`

Uses the AWS SDK v2 default credential chain:

1. `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` environment variables
2. `AWS_PROFILE` → named profile in `~/.aws/credentials`
3. Default profile in `~/.aws/credentials`
4. IAM instance role (EC2, ECS task role, Lambda execution role)

Required IAM permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ce:GetCostAndUsage",
        "ce:GetCostForecast"
      ],
      "Resource": "*"
    }
  ]
}
```

Cost Explorer is a global service — region configuration does not affect it.

---

## What the CLI does NOT do

- **No outbound network calls from the analysis modules.** Log analysis, doc search, SAST, secret scanning, and infra validation make no network requests. They work entirely on local files.
- **No telemetry.** The CLI collects nothing. No usage data, no error reporting to an external service.
- **No modification of resources.** All commands are read-only. `annave cleanup scan` is dry-run by default and does not delete resources even with `--dry-run=false` — it only reports what it finds.
- **No shell injection.** `annave infra validate` shells out to `terraform` and `helm` with fixed argument lists constructed from validated flag values. User-supplied paths are passed as arguments, not interpolated into shell strings.

---

## Responsible disclosure

If you find a security vulnerability in ANNÁVE CLI, please report it privately before opening a public issue.

**Contact:** contact@annave.tech

Include:
- A description of the vulnerability
- Steps to reproduce
- The version of `annave` affected (`annave version`)
- Your assessment of the impact

We will respond within 72 hours and aim to release a fix within 14 days of confirmation.
