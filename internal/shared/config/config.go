// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package config

import (
	_ "embed"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed limits.yaml
var limitsYAML []byte

// Limits holds per-module operational limits.
type Limits struct {
	Log     LogLimits     `yaml:"log"`
	Health  HealthLimits  `yaml:"health"`
	Doc     DocLimits     `yaml:"doc"`
	Cleanup CleanupLimits `yaml:"cleanup"`
}

// LogLimits controls resource limits for the log module.
type LogLimits struct {
	MaxLines int `yaml:"max_lines"`
}

// HealthLimits controls resource limits for the health module.
type HealthLimits struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxTargets     int `yaml:"max_targets"`
}

// DocLimits controls resource limits for the doc module.
type DocLimits struct {
	MaxFileSizeMB int `yaml:"max_file_size_mb"`
	MaxResults    int `yaml:"max_results"`
	MaxDepth      int `yaml:"max_depth"`
}

// CleanupLimits controls resource limits for the cleanup module.
type CleanupLimits struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxResources   int `yaml:"max_resources"`
}

// App holds the loaded application configuration.
var App struct {
	Limits Limits
}

func init() {
	if err := yaml.Unmarshal(limitsYAML, &App.Limits); err != nil {
		panic("tend: failed to load limits config: " + err.Error())
	}
}

// Env returns an environment variable or a fallback default.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
