<!--
title:       tend: Architecture
description: Hexagonal architecture overview, module structure, port interfaces,
             package map, and instructions for adding a new module or rule.
version:     0.1.0
created:     2026-05-16
updated:     2026-10-15
-->

# Architecture

## Design pattern

The CLI uses **hexagonal architecture** (ports and adapters). The domain core of each module (its data models and business logic) has no knowledge of Cobra, terminal output, or any specific external API. All delivery and infrastructure concerns are adapters that connect to the core through formal interfaces (ports).

This means the same analysis logic can be driven by a CLI flag, a test, or any future adapter without touching the domain core.

```
+---------------------------------------------------------+
|  CLI layer: cmd/tend/cmd/                               |
|  flag parsing, output formatting, error display         |
+---------------------------------------------------------+
                      |
                      | calls port interface
                      v
+---------------------------------------------------------+
|  Domain core: internal/<module>/domain/                 |
|  data models, value types, no external dependencies     |
|                                                         |
|  Port interface: internal/<module>/port/                |
|  one interface per module, implemented by the adapter   |
+---------------------------------------------------------+
                      |
                      | implemented by
                      v
+---------------------------------------------------------+
|  Adapter: internal/<module>/<adapter>/                  |
|  external calls: k8s API, AWS SDK, filesystem, shell    |
+---------------------------------------------------------+
```

Adding a new output format: implement a new formatter in `internal/shared/output/` and wire it to `ParseFormat`. Nothing in the domain or adapters changes.

Adding a new provider for an existing module: implement the port interface in a new adapter file and add a routing case in the orchestrator. The CLI layer is unchanged.

---

## Package map

```
github.com/annavetech/tend
|-- cmd/
|   `-- tend/
|       |-- main.go                   # entry point, resolves the version, exits with cmd.Execute's code
|       `-- cmd/
|           |-- root.go               # root command, version subcommand
|           |-- log.go                # parent: tend log
|           |-- log_analyze.go        # tend log analyze
|           |-- health.go             # parent: tend health
|           |-- health_check.go       # tend health check
|           |-- doc.go                # parent: tend doc
|           |-- doc_search.go         # tend doc search
|           |-- cleanup.go            # parent: tend cleanup
|           |-- cleanup_scan.go       # tend cleanup scan
|           |-- security.go           # parent: tend security
|           |-- security_audit.go     # tend security audit
|           |-- infra.go              # parent: tend infra
|           |-- infra_validate.go     # tend infra validate
|           |-- cost.go               # parent: tend cost
|           `-- cost_scan.go          # tend cost scan
|-- internal/
|   |-- shared/
|   |   |-- errors/                   # Error, error codes, stage constants
|   |   |-- output/                   # Printer, Format, plain/json/table rendering
|   |   `-- config/                   # embedded limits.yaml
|   |-- log/
|   |   |-- domain/                   # LogEntry, Anomaly, Finding, AnalysisReport
|   |   |-- port/                     # Analyzer interface
|   |   `-- analyzer/                 # parser, patterns, spikes, clusters, ranker
|   |-- health/
|   |   |-- domain/                   # ServiceTarget, HealthResult, CheckType, HealthReport
|   |   |-- port/                     # Checker interface
|   |   `-- checker/                  # http, tcp, dns, chain, runner
|   |-- doc/
|   |   |-- domain/                   # DocFile, SearchResult, Index, SearchQuery
|   |   |-- port/                     # Searcher interface
|   |   `-- searcher/                 # indexer, search, open, formats
|   |-- cleanup/
|   |   |-- domain/                   # K8sResource, IdleResource, CleanupPlan
|   |   |-- port/                     # Scanner interface
|   |   `-- scanner/                  # client, pods, pvcs, configmaps, namespaces, dryrun
|   |-- security/
|   |   |-- domain/                   # Finding, Severity, AuditType, AuditReport
|   |   |-- port/                     # Auditor interface
|   |   |-- auditor/                  # stub Auditor, not used by the CLI
|   |   `-- scanner/
|   |       |-- rules/                # shared Rule type and MustCompile
|   |       |-- secrets.go            # 12 secret patterns, file tree walk
|   |       |-- sast_go.go            # 10 Go SAST rules
|   |       |-- sast_ts.go            # 5 TypeScript SAST rules
|   |       |-- k8s_live.go           # live cluster checks via client-go
|   |       |-- k8s_local.go          # local YAML manifest checks
|   |       |-- walkroot.go           # shared file tree walk helpers
|   |       `-- scanner.go            # orchestrator, concurrent dispatch
|   |-- infra/
|   |   |-- domain/                   # ValidationIssue, ValidationResult, severity constants
|   |   |-- port/                     # Validator interface
|   |   `-- validator/                # terraform, helm, k8s_yaml, rootfile, runner, validator (orchestrator)
|   `-- cost/
|       |-- domain/                   # CostRecord, CostAnomaly, CostReport
|       |-- port/                     # CostAnalyzer interface
|       `-- analyzer/                 # aws (full), gcp (stub), azure (stub), analyzer (router)
`-- internal/shared/config/
    `-- limits.yaml                   # per-module limits embedded at build time
```

---

## Module anatomy

Every module follows the same three-layer structure:

### 1. Domain (`internal/<module>/domain/domain.go`)

Pure Go types. No imports from external packages or other internal packages (except `internal/shared` for primitive types). This is the data contract between the port and the adapter.

```go
type AnalysisReport struct {
    File        string
    Format      LogFormat
    TotalLines  int
    ParsedLines int
    TimeRange   TimeRange
    Findings    []Finding
    ParseErrors int
    Truncated   int
}
```

### 2. Port (`internal/<module>/port/port.go`)

A single interface that the adapter implements and the CLI calls. One interface per module.

```go
type Analyzer interface {
    Analyze(ctx context.Context, r io.Reader, filename string, opts AnalyzeOptions) (*domain.AnalysisReport, error)
}
```

### 3. Adapter (`internal/<module>/<adapter>/`)

Implements the port interface. May call external APIs, shell out to CLIs, or walk the filesystem. The domain core never sees these details.

---

## How to add a new module

1. **Create `internal/<module>/domain/domain.go`** with the result types for your module. No external dependencies.

2. **Create `internal/<module>/port/port.go`** with a single interface:
   ```go
   type MyAnalyzer interface {
       Analyze(ctx context.Context, opts AnalyzeOptions) (*domain.MyReport, error)
   }
   ```

3. **Create `internal/<module>/<adapter>/`** implementing the interface. Return wrapped Go errors with context; use `internal/shared/errors` only for an error that needs a code, such as `ERR_NOT_IMPLEMENTED`.

4. **Create `cmd/tend/cmd/<module>.go`**, the parent command:
   ```go
   var myCmd = &cobra.Command{Use: "mymodule", Short: "..."}
   func init() { rootCmd.AddCommand(myCmd) }
   ```

5. **Create `cmd/tend/cmd/<module>_<subcommand>.go`**, the subcommand:
   - Parse `--format` with `output.ParseFormat`
   - Call the port
   - Switch on `p.Format()` and call the appropriate printer

6. **Add limits** to `internal/shared/config/limits.yaml` for any configurable thresholds.

---

## How to add a new security or infra rule

### Adding a secret scanning pattern (`internal/security/scanner/secrets.go`)

Add a new `Rule` to the `secretRules` slice:

```go
{
    ID:          "SECRET013",
    Title:       "My Token",
    Severity:    domain.SeverityHigh,
    Remediation: "Move to environment variable or secrets manager",
    Pattern:     rules.MustCompile(`mytoken_[A-Za-z0-9]{32}`),
},
```

### Adding a SAST rule (`internal/security/scanner/sast_go.go` or `sast_ts.go`)

Add a new `Rule` to the `goRules` or `tsRules` slice with a `FileExts` restriction if needed:

```go
{
    ID:       "GO011",
    Title:    "Insecure TLS skip verify",
    Severity: domain.SeverityHigh,
    Pattern:  rules.MustCompile(`InsecureSkipVerify:\s*true`),
    FileExts: []string{".go"},
},
```

### Adding a Kubernetes security check

For local YAML (`internal/security/scanner/k8s_local.go`) and live cluster (`internal/security/scanner/k8s_live.go`): add a new `domain.Finding` to the slice returned by `checkManifestDoc` or `checkPodSecurity`, or by `checkLocalContainer` or `checkLiveContainer` for a per-container rule. Use the next `K8SXXX` rule ID in sequence.

---

## Error handling

Most errors are wrapped Go errors (`fmt.Errorf("...: %w", err)`) and print as plain text. An error that needs a machine-readable code uses `Error` from `internal/shared/errors`; today only the GCP and Azure cost stubs return one, with `ERR_NOT_IMPLEMENTED`:

```go
type Error struct {
    Code    string `json:"code"`
    Stage   Stage  `json:"stage"`
    Message string `json:"message"`
}
```

The package defines the stages `input`, `analysis`, `output`, `runtime`, `network`, `index` and the helpers `New`, `Newf`, `NotImplemented`, `InvalidInput`, and `IOFailure`:
```go
shared.New(shared.ErrCodeNotImplemented, shared.StageRuntime, "GCP cost analysis is not implemented yet. Use --provider aws.")
```

`Error.Error()` formats as `[<stage>/<code>] <message>`. The root command prints every error as one `error: <text>` line on stderr, or as `{"error":"<text>"}` on stdout in JSON mode, and exits `1`.
