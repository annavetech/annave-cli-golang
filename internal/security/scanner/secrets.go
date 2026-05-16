// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"bufio"
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/scanner/rules"
)

var secretRules = []rules.Rule{
	{
		ID:          "SECRET001",
		Title:       "AWS Access Key ID",
		Severity:    domain.SeverityCritical,
		Pattern:     rules.MustCompile(`AKIA[0-9A-Z]{16}`),
		Remediation: "Remove the key immediately, rotate it in AWS IAM, and use environment variables or IAM roles.",
	},
	{
		ID:          "SECRET002",
		Title:       "AWS Secret Access Key",
		Severity:    domain.SeverityCritical,
		Pattern:     rules.MustCompile(`(?i)aws.{0,30}secret.{0,10}[=:]\s*["']?[A-Za-z0-9+/]{40}["']?`),
		Remediation: "Remove the key immediately, rotate it in AWS IAM, and use environment variables or IAM roles.",
	},
	{
		ID:          "SECRET003",
		Title:       "Private key header",
		Severity:    domain.SeverityCritical,
		Pattern:     rules.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
		Remediation: "Never commit private keys. Use a secrets manager or environment variables.",
	},
	{
		ID:          "SECRET004",
		Title:       "GitHub token",
		Severity:    domain.SeverityHigh,
		Pattern:     rules.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`),
		Remediation: "Revoke the token immediately on GitHub and generate a new one.",
	},
	{
		ID:          "SECRET005",
		Title:       "Slack token",
		Severity:    domain.SeverityHigh,
		Pattern:     rules.MustCompile(`xox[baprs]-[0-9]{8,}-[0-9A-Za-z\-]+`),
		Remediation: "Revoke the token in Slack app settings and regenerate it.",
	},
	{
		ID:          "SECRET006",
		Title:       "JWT token",
		Severity:    domain.SeverityHigh,
		Pattern:     rules.MustCompile(`eyJ[A-Za-z0-9_\-]{10,}\.eyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}`),
		Remediation: "Do not hardcode JWT tokens. Issue them at runtime and store securely.",
	},
	{
		ID:          "SECRET007",
		Title:       "Database URL with credentials",
		Severity:    domain.SeverityHigh,
		Pattern:     rules.MustCompile(`(?i)(?:postgres|postgresql|mysql|mongodb|redis|amqp)://[^:\s]{1,64}:[^@\s]{6,}@`),
		Remediation: "Move database credentials to environment variables or a secrets manager.",
	},
	{
		ID:          "SECRET008",
		Title:       "Hardcoded password or secret",
		Severity:    domain.SeverityHigh,
		Pattern:     rules.MustCompile(`(?i)(?:password|passwd|pwd|secret|token)\s*(?::?=)\s*"[^"]{8,}"`),
		Remediation: "Use environment variables (os.Getenv) or a secrets manager. Never hardcode credentials.",
	},
	{
		ID:          "SECRET009",
		Title:       "Generic API key",
		Severity:    domain.SeverityMedium,
		Pattern:     rules.MustCompile(`(?i)(?:api[_-]?key|apikey)\s*[=:]\s*["']?[A-Za-z0-9_\-]{20,}["']?`),
		Remediation: "Move API keys to environment variables and add them to .gitignore.",
	},
	{
		ID:          "SECRET010",
		Title:       "Stripe secret key",
		Severity:    domain.SeverityCritical,
		Pattern:     rules.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`),
		Remediation: "Revoke the key immediately on the Stripe dashboard and rotate it.",
	},
	{
		ID:          "SECRET011",
		Title:       "SendGrid API key",
		Severity:    domain.SeverityHigh,
		Pattern:     rules.MustCompile(`SG\.[A-Za-z0-9_\-]{22}\.[A-Za-z0-9_\-]{43}`),
		Remediation: "Revoke the key in SendGrid settings and regenerate it.",
	},
	{
		ID:          "SECRET012",
		Title:       "GCP service account key",
		Severity:    domain.SeverityCritical,
		Pattern:     rules.MustCompile(`"type"\s*:\s*"service_account"`),
		Remediation: "Never commit service account JSON files. Use Workload Identity or environment variables.",
	},
}

// skipSecretDirs are directories that contain no user-committed source files.
var skipSecretDirs = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	".git": true, ".angular": true, "__pycache__": true, ".next": true,
	"coverage": true, ".cache": true,
}

const maxSecretFileSize = 5 * 1024 * 1024 // 5 MB

// scanSecrets walks root and matches all secret patterns against every non-binary file.
func scanSecrets(ctx context.Context, root string) ([]domain.Finding, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
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

		info, err := d.Info()
		if err != nil || info.Size() == 0 || info.Size() > maxSecretFileSize {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		fileFindings, err := scanFileForSecrets(path, rel)
		if err != nil {
			return nil // skip unreadable files
		}
		findings = append(findings, fileFindings...)
		return nil
	})

	return findings, err
}

// scanFileForSecrets applies all secret rules to a single file.
func scanFileForSecrets(path, relPath string) ([]domain.Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Binary detection: read first 512 bytes and check for null bytes.
	header := make([]byte, 512)
	n, _ := f.Read(header)
	if bytes.IndexByte(header[:n], 0) >= 0 {
		return nil, nil // binary file, skip
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	var findings []domain.Finding

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, rule := range secretRules {
			if !rule.MatchesExt(ext) {
				continue
			}
			if loc := rule.Pattern.FindStringIndex(line); loc != nil {
				match := line[loc[0]:loc[1]]
				findings = append(findings, domain.Finding{
					ID:          rule.ID,
					Title:       rule.Title,
					Severity:    rule.Severity,
					File:        relPath,
					Line:        lineNum,
					Detail:      redact(match),
					Remediation: rule.Remediation,
				})
			}
		}
	}
	return findings, scanner.Err()
}

// redact shows the first 12 characters of a match then masks the rest.
func redact(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12] + strings.Repeat("*", min(len(s)-12, 8)) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
