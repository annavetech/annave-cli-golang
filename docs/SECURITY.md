<!--
title:       tend: Security Notes
description: What the security audit checks, known limitations and false positives,
             how authentication works for live checks, and responsible disclosure.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Security Notes

---

## What `tend security audit` checks

The audit command has four scan types. Each targets a different attack surface.

### Secret scanning (`--type secrets`)

Walks the file tree and matches 12 regex patterns against every line of every non-binary file. The patterns are designed to find credentials that were accidentally committed, not credentials that are correctly stored in a secrets manager.

**What it scans:**
- All files in the target directory, recursively
- Files in `.env`, `.env.backup`, `.env.local`, and similar names
- YAML and JSON configuration files
- Shell scripts and Makefiles

**What it deliberately skips:**
- Binary files (detected by a zero byte in the first 512 bytes of the file)
- Symlinks, empty files, and files over 5 MB
- Generated directories: `node_modules`, `vendor`, `dist`, `build`, `.git`, `.angular`, `__pycache__`, `.next`, `coverage`, `.cache`

**Critical design choice: `.gitignore` is ignored.** The scanner does not read or respect `.gitignore`. The reason: `.env` files are in `.gitignore` precisely because they contain secrets. A secret scanner that skips `.gitignore` entries would miss the most common source of leaked credentials. To scan only committed files or git history, use a git-aware tool like `truffleHog` in addition to this scanner.

**Secret values in output are redacted.** The first 12 characters of a matched secret are shown, followed by up to 8 asterisks and `...`. This is enough to identify which credential was found without logging the full value.

### SAST (`--type sast`)

Line-by-line regex pattern matching for security-relevant code patterns. It is not an AST-based analysis. This is a practical trade-off for a developer CLI.

**Known limitation: false positives on rule definitions.** If you run `tend security audit --type sast .` against this repository, the SAST scanner will flag its own rule definition strings (the regex patterns contain the strings they match). This is expected and harmless. Run SAST against your project's source, not against this tool.

**Go rules (10, `.go` files):** `GO001` command injection (`exec.Command` or `exec.CommandContext` with `fmt.Sprintf` in the call), `GO002` SQL injection (`Query`, `QueryRow`, `Exec`, `QueryContext`, `ExecContext`, or `Prepare` with `fmt.Sprintf` or `+` concatenation), `GO003` path traversal (`os.Open`, `os.ReadFile`, `ioutil.ReadFile`, `os.Stat`, or `os.Remove` with `r.URL`, `r.Form`, `r.PostForm`, `r.Header`, or `r.PathValue`), `GO004` SSRF (`http.Get`, `http.Post`, or `http.NewRequest` with a `r.`, `req.`, or `request.` value), `GO005` weak random (`"math/rand"` import), `GO006` weak crypto (`"crypto/md5"` or `"crypto/sha1"` import), `GO007` plaintext HTTP server (any `http.ListenAndServe(` call), `GO008` unbounded read (any `io.ReadAll(` or `ioutil.ReadAll(` call), `GO009` `"unsafe"` import, `GO010` hardcoded credential (`password`, `passwd`, `secret`, `token`, `api_key`, or `auth_key` assigned a string literal of 6 or more characters).

**TypeScript and JavaScript rules (5, `.ts`, `.tsx`, `.js`, `.jsx` files):** `TS001` innerHTML assignment, `TS002` `dangerouslySetInnerHTML`, `TS003` `eval()`, `TS004` `document.write()`, `TS005` `localStorage.setItem` with a token, password, secret, auth, or key name.

**What it does not check:** Call graphs, data flow, taint analysis, or cross-file analysis. A `db.Query` call that receives its argument from a function parameter will not be flagged unless the pattern matches at the call site. For deeper analysis, use a dedicated SAST tool (Semgrep, CodeQL).

### Kubernetes live checks (`--type k8s-live`)

Lists pods from the cluster API, in `--namespace` or in all namespaces. Does not modify any resource. Requires `list` permission on pods in the namespace, or cluster-wide without `--namespace`.

**Checked rules:** `hostNetwork: true` (K8S001), `hostPID: true` (K8S002), `hostPath` volume (K8S003), privileged container (K8S004), container that may run as root (K8S005), container without resource limits (K8S006), container without a readiness probe (K8S007).

K8S005 fires when the effective `runAsUser` (container value over pod value) is 0, or when no `runAsUser` is set and `runAsNonRoot` is not `true`. Init and ephemeral containers are checked too. K8S006 does not apply to ephemeral containers, and K8S007 applies only to regular containers.

### Kubernetes local manifest checks (`--type k8s-local`)

Parses YAML files from disk. Does not require a cluster connection. Handles multi-document YAML files (documents separated by `---`).

Reads `.yaml` and `.yml` files and checks Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet, Job, and CronJob specs. Same rules as `k8s-live` plus K8S008: image tag is `:latest` or missing (non-deterministic deployments). A file that cannot be read is named in a `warning:` line, and a document that does not parse is skipped.

---

## Authentication for live checks

### `tend security audit --type k8s-live`

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
  verbs: ["list"]
```

### `tend cost scan`

Uses the AWS SDK v2 default credential chain:

1. `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY` environment variables
2. `AWS_PROFILE`, which names a profile in `~/.aws/credentials`
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

Cost Explorer is a global service, so region configuration does not affect it.

---

## What the CLI does NOT do

- **No outbound network calls from the analysis modules.** Log analysis, doc search, SAST, secret scanning, local Kubernetes checks, and infra validation of Terraform plans, charts, and manifests make no network requests. They work entirely on local files. `tend infra validate` with no target runs `helm list`, which contacts the cluster.
- **No telemetry.** The CLI collects nothing. No usage data, no error reporting to an external service.
- **No modification of resources.** All commands are read-only. `tend cleanup scan` is dry-run by default and does not delete resources even with `--dry-run=false`. It only reports what it finds.
- **No shell injection.** `tend infra validate` shells out to `terraform` and `helm` with fixed argument lists constructed from validated flag values. User-supplied paths are passed as arguments, not interpolated into shell strings.

---

## Responsible disclosure

If you find a security vulnerability in tend, please report it privately before opening a public issue.

**Contact:** [GitHub security advisories](https://github.com/annavetech/tend/security/advisories/new)

Include:
- A description of the vulnerability
- Steps to reproduce
- The version of `tend` affected (`tend version`)
- Your assessment of the impact

We will respond within 72 hours and aim to release a fix within 14 days of confirmation.
