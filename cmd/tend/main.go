// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package main

import (
	"os"
	"runtime/debug"
	"strings"

	"github.com/annavetech/tend/cmd/tend/cmd"
)

// version is set at release time with -ldflags "-X main.version=<version>".
var version = "dev"

func main() {
	os.Exit(cmd.Execute(resolveVersion()))
}

// resolveVersion returns the ldflags version, else the module version stamped by go install.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return version
}
