// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixture writes testdata/name into dir and returns its path.
func copyFixture(t *testing.T, dir, name string) string {
	t.Helper()
	src, err := os.OpenRoot("testdata")
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestRoot(t, src)
	data, err := src.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestRoot(t, dst)
	if err := dst.WriteFile(name, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, name)
}

func closeTestRoot(t *testing.T, root *os.Root) {
	t.Helper()
	if err := root.Close(); err != nil {
		t.Error(err)
	}
}

func TestWalkManifestsDoesNotFollowSymlinkOutOfRoot(t *testing.T) {
	outside := copyFixture(t, t.TempDir(), "cronjob-latest.yaml")
	root := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link.yaml")); err != nil {
		t.Fatal(err)
	}
	copyFixture(t, root, "cronjob-latest.yaml")

	issues, warnings, err := walkAndCheckManifests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "skipped link.yaml") {
		t.Fatalf("want one warning naming link.yaml, got %q", warnings)
	}
	for _, is := range issues {
		if strings.Contains(is.Resource, "link.yaml") {
			t.Fatalf("read a file outside the root through a symlink: %+v", is)
		}
	}
	if !strings.Contains(rules(issues), "K8S104") {
		t.Fatalf("want K8S104 from the file inside the root, got %q", rules(issues))
	}
	res, err := validateK8sYAML(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("want the warning on the result, got %q", res.Warnings)
	}
}

func TestReadFileThroughSymlinkInSameDir(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "cronjob-latest.yaml")
	link := filepath.Join(dir, "link.yaml")
	if err := os.Symlink("cronjob-latest.yaml", link); err != nil {
		t.Fatal(err)
	}
	res, err := validateK8sYAML(link)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rules(res.Issues), "K8S104") {
		t.Fatalf("want K8S104, got %q", rules(res.Issues))
	}
}

func TestExplicitTargetSymlinkToOtherDirValidates(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	links := filepath.Join(base, "links")
	for _, d := range []string{real, links} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	copyFixture(t, real, "cronjob-latest.yaml")
	link := filepath.Join(links, "pod.yaml")
	if err := os.Symlink(filepath.Join("..", "real", "cronjob-latest.yaml"), link); err != nil {
		t.Fatal(err)
	}
	res, err := validateK8sYAML(link)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rules(res.Issues), "K8S104") {
		t.Fatalf("want K8S104 through the symlinked target, got %q", rules(res.Issues))
	}
}

func TestExplicitTerraformPlanSymlinkToOtherDirReads(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "real"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(base, "links"), 0o700); err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenRoot(filepath.Join(base, "real"))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestRoot(t, dst)
	if err := dst.WriteFile("plan.json", []byte(`{"resource_changes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "links", "plan.json")
	if err := os.Symlink(filepath.Join("..", "real", "plan.json"), link); err != nil {
		t.Fatal(err)
	}
	data, err := loadTerraformJSON(t.Context(), fakeRunner{t: t}, link)
	if err != nil || string(data) != `{"resource_changes":[]}` {
		t.Fatalf("data = %q, err = %v", data, err)
	}
}
