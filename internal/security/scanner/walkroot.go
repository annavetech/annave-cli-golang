// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: MIT
package scanner

import (
	"fmt"
	"os"
	"path/filepath"
)

// openWalkRoot opens an os.Root for a walk over absRoot, so a symlink cannot lead a read outside it.
// A file root opens its directory. A nil root with a nil error means absRoot cannot be read.
func openWalkRoot(absRoot string) (*os.Root, string, error) {
	info, err := os.Lstat(absRoot)
	if err != nil {
		return nil, "", nil
	}
	dir := absRoot
	if !info.IsDir() {
		dir = filepath.Dir(absRoot)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", err
	}
	return root, dir, nil
}

// skippedFileWarning names a file that was not scanned and the reason.
func skippedFileWarning(relPath string, err error) string {
	return fmt.Sprintf("skipped %s: %v", relPath, err)
}

// closeRoot closes root and keeps the first error.
func closeRoot(root *os.Root, err *error) {
	if cerr := root.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}

// closeFile closes f and keeps the first error.
func closeFile(f *os.File, err *error) {
	if cerr := f.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}
