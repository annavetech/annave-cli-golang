// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package rules

import (
	"regexp"

	"annave.tech/cli/internal/security/domain"
)

// Rule defines a single detection pattern for secrets or SAST scanning.
type Rule struct {
	ID          string
	Title       string
	Severity    domain.Severity
	Remediation string
	Pattern     *regexp.Regexp
	FileExts    []string // nil = all files; [".go"] = Go files only
}

// MatchesExt reports whether this rule applies to a file with the given extension.
func (r Rule) MatchesExt(ext string) bool {
	if len(r.FileExts) == 0 {
		return true
	}
	for _, e := range r.FileExts {
		if e == ext {
			return true
		}
	}
	return false
}

// MustCompile wraps regexp.MustCompile for inline rule definitions.
func MustCompile(expr string) *regexp.Regexp {
	return regexp.MustCompile(expr)
}
