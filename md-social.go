// frontmatter-cli — tiny zero-deps CLI to read/modify Markdown frontmatter in Go
//
// Features:
//  - Detect YAML frontmatter at the top of a Markdown file (--- ... ---)
//  - Parse a small, practical subset of YAML (strings/numbers/booleans/arrays, simple block scalars)
//  - Get/Set/Delete keys using dot paths and write changes back to the file
//  - Initialize frontmatter if missing
//
// Usage:
//   go run frontmatter-cli.go show ./post.md
//   go run frontmatter-cli.go get ./post.md title
//   go run frontmatter-cli.go set ./post.md title "My Post"
//   go run frontmatter-cli.go set ./post.md tags "[go, cli]"
//   go run frontmatter-cli.go del ./post.md draft
//   go run frontmatter-cli.go init ./post.md
//
// Notes:
//  - No third-party dependencies; minimal YAML reader/writer for common frontmatter.
//  - Not fully YAML-spec compliant; optimized for typical blog/docs frontmatter.
//  - Keys are written back in sorted order for stable diffs.

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

type FMValue = any

type Parsed struct {
	Frontmatter    map[string]any
	Body           string
	HasFrontmatter bool
	RawBlock       string // without --- markers
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	file := os.Args[1]
	if _, err := os.Stat(file); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("file not found: %s", file)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	parsed := parseMDContents(b)

	enc := yaml.NewEncoder(os.Stdout)
	//enc.SetIndent("", "  ")
	if parsed.Frontmatter == nil {
		parsed.Frontmatter = map[string]FMValue{}
	}

	parsed.Frontmatter["social"] = map[string]string{"bluesky": "bleh"}
	return enc.Encode(parsed.Frontmatter)
}
