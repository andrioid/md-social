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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type FMValue = any

var baseURL string = "https://andri.dk/blog"
var debug = false

var (
	ErrInvalidFile = errors.New("invalid markdown file")
	ErrSkipped     = errors.New("file skipped")
)

var publishers []Publisher

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	fmt.Println("Connecting to publishers...")
	publishers = []Publisher{NewBluesky(ctx)}
	fmt.Println("Connected")

	if os.Getenv("DEBUG") == "1" {
		debug = true
	}
	if os.Getenv("MD_BASE_URL") != "" {
		baseURL = os.Getenv("MD_BASE_URL")
	}

	dir := os.Args[1]
	if stat, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) || !stat.IsDir() {
		return fmt.Errorf("directory not found: %s", dir)
	}

	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		// Skip if extension isn't MD
		ext := filepath.Ext(path)
		if ext != ".md" {
			fmt.Println("unsupported ext", ext, path)
			return nil
		}
		err = handleFile(ctx, path, dir)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Handles a single file
// 1. Parse it into Parsed
// 2. Extract title
// 3. Extract URL
// 4. Create social post if we have credentials
// 5. Update file with social URL if succeeded
func handleFile(ctx context.Context, file string, prefix string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	md := Parse(b, file, prefix)
	if md == nil {
		fmt.Println("skipping, parser returned nil", file, prefix)
		return nil
	}

	p := md.GetPost()

	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// Don't publish if the post is old
	if !p.date.IsZero() && p.date.Before(oneWeekAgo) {
		fmt.Println("Skipping old record from", p.date)
		return nil
	}

	// Don't publish if there's no title or URL
	if p.title == "" || p.url == "" {
		return nil
	}

	for _, publisher := range publishers {
		sl := md.GetSocial(publisher.PublisherID())
		if sl != "" {
			fmt.Println("Skipping existing record")
			continue // Existing record
		}
		u, err := publisher.Publish(ctx, md)
		if err != nil {
			return err
		}
		fmt.Printf("Posted to %s: %s\n", publisher.PublisherID(), u)
		md.SetSocial(publisher.PublisherID(), u)
	}

	//fmt.Println("pre-save", md)

	if md.PendingWrite {
		wfile, err := os.Create(file)
		if err != nil {
			return err
		}
		defer wfile.Close()
		_, err = md.WriteTo(wfile)
		if err != nil {
			return err
		}
	}
	return nil
}
