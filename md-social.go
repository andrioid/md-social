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
	if os.Getenv("DEBUG") == "1" {
		debug = true
	}

	ctx := context.Background()

	// Publishers
	fmt.Println("Connecting to publishers...")

	if debug {
		publishers = []Publisher{}
	}
	publishers = []Publisher{NewBluesky(ctx)}
	fmt.Println("Connected")

	if os.Getenv("MD_BASE_URL") != "" {
		baseURL = os.Getenv("MD_BASE_URL")
	}

	// File handling
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
		//fmt.Println("Skipping old record from", p.date)
		return nil
	}

	// Don't publish if there's no title or URL
	if p.title == "" || p.url == "" {
		return nil
	}

	if len(publishers) == 0 {
		fmt.Printf("Dryrun: %s\n", md.Filename)
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
