// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestK8sLocalDoesNotFollowSymlinkOutOfRoot(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "cronjob-privileged.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	outsideDir := t.TempDir()
	writeFile(t, outsideDir, "outside.yaml", string(data))
	outside := filepath.Join(outsideDir, "outside.yaml")
	root := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link.yaml")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "inside.yaml", string(data))

	findings, warnings, err := scanK8sLocal(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "skipped link.yaml") {
		t.Fatalf("want one warning naming link.yaml, got %q", warnings)
	}
	for _, f := range findings {
		if strings.Contains(f.File, "link.yaml") {
			t.Fatalf("read a file outside the root through a symlink: %+v", f)
		}
	}
	if len(findByID(findings, "K8S004")) != 1 {
		t.Fatalf("want K8S004 from inside.yaml, got %+v", findings)
	}
}

func TestSecretsScansSingleFileRoot(t *testing.T) {
	dir := t.TempDir()
	token := "ghp_" + strings.Repeat("A1b2", 9)
	writeFile(t, dir, "a.js", "const t = \""+token+"\";\n")

	findings, warnings, err := scanSecrets(context.Background(), filepath.Join(dir, "a.js"))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("err=%v warnings=%v", err, warnings)
	}
	if len(findByID(findings, "SECRET004")) != 1 {
		t.Fatalf("want SECRET004 from the file root, got %+v", findings)
	}
}

func TestScansOfMissingRootReturnNothing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if f, w, err := scanSecrets(context.Background(), missing); err != nil || len(f)+len(w) != 0 {
		t.Fatalf("scanSecrets: findings=%v warnings=%v err=%v", f, w, err)
	}
	if f, w, err := scanK8sLocal(context.Background(), missing); err != nil || len(f)+len(w) != 0 {
		t.Fatalf("scanK8sLocal: findings=%v warnings=%v err=%v", f, w, err)
	}
}
