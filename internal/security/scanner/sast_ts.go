// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"

	"annave.tech/cli/internal/security/domain"
	"annave.tech/cli/internal/security/scanner/rules"
)

var tsRules = []rules.Rule{
	{
		ID:       "TS001",
		Title:    "XSS: innerHTML assignment",
		Severity: domain.SeverityHigh,
		Pattern:  rules.MustCompile(`\.innerHTML\s*=[^=]`),
		FileExts: []string{".ts", ".tsx", ".js", ".jsx"},
		Remediation: "Use textContent instead of innerHTML, or sanitise HTML with a library " +
			"like DOMPurify before assigning to innerHTML.",
	},
	{
		ID:       "TS002",
		Title:    "XSS: dangerouslySetInnerHTML (React/Angular)",
		Severity: domain.SeverityHigh,
		Pattern:  rules.MustCompile(`dangerouslySetInnerHTML\s*=`),
		FileExts: []string{".ts", ".tsx", ".js", ".jsx"},
		Remediation: "Sanitise all HTML passed to dangerouslySetInnerHTML with DOMPurify. " +
			"Prefer rendering components over raw HTML where possible.",
	},
	{
		ID:       "TS003",
		Title:    "Code injection: eval() call",
		Severity: domain.SeverityCritical,
		Pattern:  rules.MustCompile(`\beval\s*\(`),
		FileExts: []string{".ts", ".tsx", ".js", ".jsx"},
		Remediation: "Never use eval(). Parse JSON with JSON.parse(), use safer alternatives " +
			"for dynamic code execution, or restructure the logic.",
	},
	{
		ID:       "TS004",
		Title:    "XSS: document.write()",
		Severity: domain.SeverityHigh,
		Pattern:  rules.MustCompile(`document\.write\s*\(`),
		FileExts: []string{".ts", ".tsx", ".js", ".jsx"},
		Remediation: "Avoid document.write(). Use DOM manipulation methods (createElement, " +
			"appendChild) or template literals with textContent instead.",
	},
	{
		ID:       "TS005",
		Title:    "Sensitive data in localStorage",
		Severity: domain.SeverityMedium,
		Pattern:  rules.MustCompile(`(?i)localStorage\.setItem\s*\([^,]*(?:token|password|secret|auth|key)`),
		FileExts: []string{".ts", ".tsx", ".js", ".jsx"},
		Remediation: "Do not store sensitive credentials in localStorage — it is accessible to " +
			"any JavaScript on the page. Use httpOnly cookies or in-memory state instead.",
	},
}

// scanSASTTypeScript runs TypeScript/JavaScript SAST rules against frontend source files.
func scanSASTTypeScript(ctx context.Context, root string) ([]domain.Finding, error) {
	return scanSASTFiles(ctx, root, tsRules, []string{".ts", ".tsx", ".js", ".jsx"})
}
