package main

import (
	"os"
	"path/filepath"
)

// readDirRecursive reads the given `dir` recursively
func readDirRecursive(dir string) ([]string, error) {
	var res []string

	f := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			res = append(res, path)
		}

		return nil
	}

	return res, filepath.Walk(dir, f)
}
