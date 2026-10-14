// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"annave.tech/cli/internal/security/domain"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := root.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func findByID(findings []domain.Finding, id string) []domain.Finding {
	var out []domain.Finding
	for _, f := range findings {
		if f.ID == id {
			out = append(out, f)
		}
	}
	return out
}

func TestSASTGoRedactsHardcodedCredential(t *testing.T) {
	const secret = "SuperSecretValue123!"
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n\nvar pass"+"word = \""+secret+"\"\n")

	findings, warnings, err := scanSASTGo(context.Background(), dir)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	got := findByID(findings, "GO010")
	if len(got) != 1 {
		t.Fatalf("want 1 GO010 finding, got %d", len(got))
	}
	if strings.Contains(got[0].Detail, secret) || strings.Contains(got[0].Detail, "Secret") {
		t.Errorf("detail leaks the credential: %q", got[0].Detail)
	}
}

func TestSASTTypeScriptRedactsLocalStorageLine(t *testing.T) {
	const value = "abc123secretvalue"
	dir := t.TempDir()
	writeFile(t, dir, "app.ts", "localStorage.setItem(\"token\", \""+value+"\");\n")

	findings, _, err := scanSASTTypeScript(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	got := findByID(findings, "TS005")
	if len(got) != 1 {
		t.Fatalf("want 1 TS005 finding, got %d", len(got))
	}
	if strings.Contains(got[0].Detail, value) {
		t.Errorf("detail leaks the stored value: %q", got[0].Detail)
	}
}

func TestSASTKeepsLineForNonSensitiveRules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.ts", "  eval(userInput);\n")

	findings, _, err := scanSASTTypeScript(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	got := findByID(findings, "TS003")
	if len(got) != 1 || got[0].Detail != "eval(userInput);" {
		t.Fatalf("want TS003 with the trimmed line, got %+v", got)
	}
}

func TestSASTFindsMatchAfterLongLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n// "+strings.Repeat("x", 70*1024)+"\nimport \"unsafe\"\n")

	findings, warnings, err := scanSASTGo(context.Background(), dir)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if len(findByID(findings, "GO009")) != 1 {
		t.Fatalf("want GO009 after a 70 KB line, got %+v", findings)
	}
}

func TestSecretsFindsTokenBeforeLongLine(t *testing.T) {
	dir := t.TempDir()
	token := "ghp_" + strings.Repeat("A1b2", 9)
	writeFile(t, dir, "bundle.js", "const t = \""+token+"\";\n"+strings.Repeat("x", 70*1024)+"\n")

	findings, warnings, err := scanSecrets(context.Background(), dir)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	got := findByID(findings, "SECRET004")
	if len(got) != 1 || got[0].Line != 1 {
		t.Fatalf("want SECRET004 on line 1, got %+v", got)
	}
}

func TestSensitiveDetailHidesWholeValue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n\nfunc f() {\n\ttoken=\"SuperSecretValue123!\"\n}\n")
	writeFile(t, dir, "app.ts", "localStorage.setItem(\"token\", v);\n")

	goFindings, _, err := scanSASTGo(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := findByID(goFindings, "GO010"); len(got) != 1 || got[0].Detail != `token="********` {
		t.Fatalf("GO010 detail: %+v", got)
	}
	tsFindings, _, err := scanSASTTypeScript(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := findByID(tsFindings, "TS005"); len(got) != 1 || got[0].Detail != "localStorage.setItem(********" {
		t.Fatalf("TS005 detail: %+v", got)
	}
}

func TestScansSkipSymlinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "sub"), filepath.Join(dir, "dirlink.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken.go")); err != nil {
		t.Fatal(err)
	}

	if _, warnings, err := scanSecrets(context.Background(), dir); err != nil || len(warnings) != 0 {
		t.Errorf("secrets: err=%v warnings=%v", err, warnings)
	}
	if _, warnings, err := scanSASTGo(context.Background(), dir); err != nil || len(warnings) != 0 {
		t.Errorf("sast: err=%v warnings=%v", err, warnings)
	}
}
