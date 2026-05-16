// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/scanner/rules"
)

var goRules = []rules.Rule{
	{
		ID:       "GO001",
		Title:    "Command injection: exec.Command with fmt.Sprintf",
		Severity: domain.SeverityCritical,
		Pattern:  rules.MustCompile(`exec\.(?:Command|CommandContext)\([^)]*fmt\.Sprintf\(`),
		FileExts: []string{".go"},
		Remediation: "Never build shell commands from user input. Pass fixed command strings with " +
			"validated arguments, or use a whitelist of allowed values.",
	},
	{
		ID:       "GO002",
		Title:    "SQL injection: query built with string concatenation",
		Severity: domain.SeverityCritical,
		Pattern:  rules.MustCompile(`\.(?:Query|QueryRow|Exec|QueryContext|ExecContext|Prepare)\((?:fmt\.Sprintf\(|[^"\n)]*\+)`),
		FileExts: []string{".go"},
		Remediation: "Use parameterised queries with placeholders ($1, ?) instead of string " +
			"concatenation or fmt.Sprintf in SQL statements.",
	},
	{
		ID:       "GO003",
		Title:    "Path traversal: file opened with HTTP request value",
		Severity: domain.SeverityHigh,
		Pattern:  rules.MustCompile(`(?:os\.Open|os\.ReadFile|ioutil\.ReadFile|os\.Stat|os\.Remove)\([^)\n]*r\.(?:URL|Form|PostForm|Header|PathValue)`),
		FileExts: []string{".go"},
		Remediation: "Sanitise file paths with filepath.Clean and filepath.Join against a " +
			"known safe base directory. Reject paths containing '..'.",
	},
	{
		ID:       "GO004",
		Title:    "SSRF: outbound HTTP request with request-derived URL",
		Severity: domain.SeverityHigh,
		Pattern:  rules.MustCompile(`http\.(?:Get|Post|NewRequest)\([^)\n]*(?:r\.|req\.|request\.)`),
		FileExts: []string{".go"},
		Remediation: "Validate and allowlist outbound URLs. Never forward user-supplied URLs " +
			"to internal services.",
	},
	{
		ID:       "GO005",
		Title:    "Weak random: math/rand used (not cryptographically secure)",
		Severity: domain.SeverityMedium,
		Pattern:  rules.MustCompile(`"math/rand"`),
		FileExts: []string{".go"},
		Remediation: "Use crypto/rand for security-sensitive randomness (tokens, passwords, " +
			"nonces). math/rand is predictable and not suitable for security contexts.",
	},
	{
		ID:       "GO006",
		Title:    "Weak crypto: MD5 or SHA-1 used",
		Severity: domain.SeverityMedium,
		Pattern:  rules.MustCompile(`"crypto/(?:md5|sha1)"`),
		FileExts: []string{".go"},
		Remediation: "MD5 and SHA-1 are broken for security use. Use SHA-256 (crypto/sha256) " +
			"or stronger. For passwords use bcrypt, argon2, or scrypt.",
	},
	{
		ID:       "GO007",
		Title:    "Plaintext HTTP server: ListenAndServe without TLS",
		Severity: domain.SeverityMedium,
		Pattern:  rules.MustCompile(`http\.ListenAndServe\(`),
		FileExts: []string{".go"},
		Remediation: "Use http.ListenAndServeTLS or terminate TLS at a reverse proxy. " +
			"Serving plain HTTP exposes credentials and data in transit.",
	},
	{
		ID:       "GO008",
		Title:    "Unbounded read: io.ReadAll without size limit",
		Severity: domain.SeverityLow,
		Pattern:  rules.MustCompile(`(?:io|ioutil)\.ReadAll\(`),
		FileExts: []string{".go"},
		Remediation: "Wrap the reader with io.LimitReader before passing to io.ReadAll to " +
			"prevent memory exhaustion from large or malicious payloads.",
	},
	{
		ID:       "GO009",
		Title:    "Unsafe package imported",
		Severity: domain.SeverityMedium,
		Pattern:  rules.MustCompile(`"unsafe"`),
		FileExts: []string{".go"},
		Remediation: "Avoid the unsafe package in production code. It bypasses Go's type safety " +
			"and can cause memory corruption. Review whether this use is necessary.",
	},
	{
		ID:       "GO010",
		Title:    "Hardcoded credential in source",
		Severity: domain.SeverityHigh,
		Pattern:  rules.MustCompile(`(?i)(?:password|passwd|secret|token|api_?key|auth_?key)\s*(?::?=)\s*"[^"]{6,}"`),
		FileExts: []string{".go"},
		Remediation: "Use os.Getenv or a secrets manager. Never hardcode credentials in source — " +
			"they will appear in version history even after deletion.",
	},
}

// scanSASTGo runs Go-specific SAST rules against all .go files under root.
func scanSASTGo(ctx context.Context, root string) ([]domain.Finding, error) {
	return scanSASTFiles(ctx, root, goRules, []string{".go"})
}

// scanSASTFiles walks root for files matching exts and applies ruleSet line by line.
func scanSASTFiles(ctx context.Context, root string, ruleSet []rules.Rule, exts []string) ([]domain.Finding, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		extSet[e] = true
	}

	var findings []domain.Finding

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if skipSecretDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !extSet[ext] {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() == 0 || info.Size() > maxSecretFileSize {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		ff, err := applyRulesToFile(path, rel, ext, ruleSet)
		if err != nil {
			return nil
		}
		findings = append(findings, ff...)
		return nil
	})
	return findings, err
}

// applyRulesToFile applies ruleSet to a single file and returns any matching findings.
func applyRulesToFile(path, relPath, ext string, ruleSet []rules.Rule) ([]domain.Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []domain.Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, rule := range ruleSet {
			if !rule.MatchesExt(ext) {
				continue
			}
			if loc := rule.Pattern.FindStringIndex(line); loc != nil {
				match := strings.TrimSpace(line)
				if len(match) > 120 {
					match = match[:117] + "..."
				}
				findings = append(findings, domain.Finding{
					ID:          rule.ID,
					Title:       rule.Title,
					Severity:    rule.Severity,
					File:        relPath,
					Line:        lineNum,
					Detail:      match,
					Remediation: rule.Remediation,
				})
			}
		}
	}
	return findings, scanner.Err()
}
