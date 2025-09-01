package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func getMarkdownFiles(dir string) ([]string, error) {
	files := []string{}

	if stat, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) || !stat.IsDir() {
		return files, fmt.Errorf("%w: failed to stat md directory '%s'", err, dir)
	}

	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return files, err
	}

	return files, nil

}
