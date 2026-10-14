// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package validator

import (
	"fmt"
	"os"
	"path/filepath"
)

// readFile reads an explicitly named target file. Symlinks are resolved first,
// so a link to a file in another directory is read from that directory's os.Root.
func readFile(path string) (data []byte, err error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	root, err := os.OpenRoot(filepath.Dir(resolved))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := root.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	data, err = root.ReadFile(filepath.Base(resolved))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}
