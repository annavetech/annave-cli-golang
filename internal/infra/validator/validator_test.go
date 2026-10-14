// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import (
	"context"
	"embed"
	"errors"
	"strings"
	"testing"

	"annave.tech/cli/internal/infra/domain"
	"annave.tech/cli/internal/infra/port"
)

//go:embed testdata
var testdata embed.FS

// fakeRunner replays recorded output keyed by the full command line.
type fakeRunner struct {
	t       *testing.T
	outputs map[string]string // command line to testdata file
	exitErr error             // returned with every output, like a non-zero exit
}

func (f fakeRunner) LookPath(file string) (string, error) { return file, nil }

func (f fakeRunner) Output(ctx context.Context, _, name string, args ...string) ([]byte, error) {
	return f.CombinedOutput(ctx, name, args...)
}

func (f fakeRunner) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	line := strings.Join(append([]string{name}, args...), " ")
	file, ok := f.outputs[line]
	if !ok {
		f.t.Fatalf("unexpected command %q", line)
	}
	data, err := testdata.ReadFile("testdata/" + file)
	if err != nil {
		f.t.Fatal(err)
	}
	return data, f.exitErr
}

func rules(issues []domain.ValidationIssue) string {
	var ids []string
	for _, i := range issues {
		ids = append(ids, i.Rule)
	}
	return strings.Join(ids, ",")
}

func TestHelmListIncludesPendingAndSuperseded(t *testing.T) {
	run := fakeRunner{t: t, outputs: map[string]string{
		"helm list -A -o json":       "helm-list-default.json",
		"helm list -A -o json --all": "helm-list-all.json",
	}}
	res, err := validateHelm(t.Context(), run, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := rules(res.Issues); got != "HELM001,HELM002" {
		t.Fatalf("got rules %q, want HELM001,HELM002", got)
	}
	if res.Passed {
		t.Error("a pending release must fail validation")
	}
}

func TestHelmLintFailureWithoutErrorLine(t *testing.T) {
	run := fakeRunner{t: t, exitErr: errors.New("exit status 1"), outputs: map[string]string{
		"helm lint /charts/missing": "helm-lint-missing-chart.txt",
	}}
	_, err := lintChart(t.Context(), run, "helm", "/charts/missing")
	if err == nil || !strings.Contains(err.Error(), "Chart.yaml") {
		t.Fatalf("want an error with helm output, got %v", err)
	}
}

func TestHelmLintErrorLinesStillReported(t *testing.T) {
	run := fakeRunner{t: t, exitErr: errors.New("exit status 1"), outputs: map[string]string{
		"helm lint /charts/web": "helm-lint-errors.txt",
	}}
	issues, err := lintChart(t.Context(), run, "helm", "/charts/web")
	if err != nil {
		t.Fatal(err)
	}
	if got := rules(issues); got != "HELM005,HELM003" {
		t.Fatalf("got rules %q, want HELM005,HELM003", got)
	}
}

func TestK8sYAMLCronJobContainers(t *testing.T) {
	res, err := validateK8sYAML("testdata/cronjob-latest.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rules(res.Issues), "K8S104") {
		t.Fatalf("want K8S104 for the CronJob container, got %q", rules(res.Issues))
	}
}

// ctxRunner records the context each command runs under.
type ctxRunner struct {
	fakeRunner
	got *context.Context
}

func (r ctxRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	*r.got = ctx
	return r.fakeRunner.Output(ctx, dir, name, args...)
}

func TestValidatePassesContextToRunner(t *testing.T) {
	type key struct{}
	var got context.Context
	run := ctxRunner{fakeRunner: fakeRunner{t: t, outputs: map[string]string{
		"helm list -A -o json --all": "helm-list-all.json",
	}}, got: &got}
	ctx := context.WithValue(t.Context(), key{}, "caller")
	v := &InfraValidator{run: run}
	if _, err := v.Validate(ctx, "", port.ValidateOptions{Type: "helm"}); err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Value(key{}) != "caller" {
		t.Fatal("helm must run under the caller's context")
	}
}
