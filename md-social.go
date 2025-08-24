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
var ogImageBackground string = ""
var ogImageAuthorImage string = ""

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

func usage() {
	fmt.Println("usage: md-social <md-directory>")
	fmt.Println("")
	fmt.Println("Supported ENV variables")
	fmt.Println("- MD_BASE_URL, BLUESKY_HANDLE*, BLUESKY_HOST, BLUESKY_APP_PASSWORD*, DEBUG")
}

func run() error {
	if os.Getenv("DEBUG") == "1" {
		debug = true
	}

	if len(os.Args) < 2 || os.Args[1] == "" {
		usage()
		os.Exit(0)
	}
	if os.Getenv("OG_IMAGE_BG") != "" {
		ogImageBackground = os.Getenv("OG_IMAGE_BG")
	}

	dir := os.Args[1]

	ctx := context.Background()

	// Publishers
	if debug {
		publishers = []Publisher{}
	} else {
		fmt.Println("Connecting to publishers...")
		publishers = []Publisher{NewBluesky(ctx)}
	}

	if os.Getenv("MD_BASE_URL") != "" {
		baseURL = os.Getenv("MD_BASE_URL")
	}

	// File handling
	if stat, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) || !stat.IsDir() {
		return fmt.Errorf("directory not found: %s", dir)
	}

	files := []string{}
	filesTotal := 0
	filesSkipped := 0
	filesPublished := 0
	filesFailed := 0

	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		filesTotal++

		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}

	for _, file := range files {
		err = handleFile(ctx, file, dir)
		if err != nil {
			// TODO: Fault tolerance
			if errors.Is(err, ErrSkipped) {
				if debug {
					fmt.Println(err)
				}

				filesSkipped++
				continue
			}
			fmt.Println(err)
			filesFailed++
		}
		filesPublished++
	}

	fmt.Printf("Found %d markdown files. %d published, %d skipped.\n", filesTotal, filesPublished, filesSkipped)

	return nil
}

// Handles a single file
// 1. Parse it into Parsed
// 2. Extract title
// 3. Extract URL
// 4. Create social post if we have credentials
// 5. Update file with social URL if succeeded
func handleFile(ctx context.Context, file string, prefix string) error {
	ext := filepath.Ext(file)
	if ext != ".md" {
		return fmt.Errorf("%w: file extension not markdown: %s", ErrSkipped, file)
	}

	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	md := Parse(b, file, prefix)
	if md == nil {
		return fmt.Errorf("%w: no frontmatter found: %s", ErrSkipped, file)
	}

	p := md.GetPost()

	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// Don't publish if the post is old
	if !p.date.IsZero() && p.date.Before(oneWeekAgo) {
		//fmt.Println("Skipping old record from", p.date)
		return fmt.Errorf("%w: post older than one week: %s", ErrSkipped, file)
	}

	// Don't publish if there's no title or URL
	if p.title == "" || p.url == "" {
		return fmt.Errorf("%w: no title or url in frontmatter: %s", ErrSkipped, file)

	}

	// MDF Processors
	ogi, err := NewOgImageGenerator(true)
	if err != nil {
		return err
	}
	for _, processor := range []MDFProcessor{ogi} {
		err = processor.Process(ctx, md)
		if err != nil {
			return err
		}
	}

	// Publishers
	for _, publisher := range publishers {
		if len(publishers) == 0 {
			return fmt.Errorf("%w: dry-run publish: %s", ErrSkipped, file)
		}

		sl := md.GetSocial(publisher.PublisherID())
		if sl != "" {
			//fmt.Println("Skipping existing record")
			continue // Existing record
		}
		u, err := publisher.Publish(ctx, md)
		if err != nil {
			return err
		}
		fmt.Printf("Posted to %s: %s\n", publisher.PublisherID(), u)
		md.SetSocial(publisher.PublisherID(), u)
	}
	if !md.PendingWrite {
		return fmt.Errorf("%w: existing record(s): %s", ErrSkipped, file)
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
