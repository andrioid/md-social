package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"
)

func parse(ctx context.Context, cmd *cli.Command) error {
	verbose = cmd.Bool("verbose")
	dir := cmd.StringArg("dir")
	if dir == "" {
		return fmt.Errorf("no directory, given: %s", dir)
	}
	dryRun = cmd.Bool("dryrun")
	baseURL := cmd.String("base-url")
	pubMaxDays := cmd.Int("publish-max-days")

	processing := []Processor{}
	publishing := []Publisher{}

	// modules
	if bsky, err := NewBluesky(ctx, cmd); err == nil {
		publishing = append(publishing, bsky)
	} else {
		if !errors.Is(err, ErrModuleSkipped) {
			return err
		}
		fmt.Println("[module] bluesky not loaded")
	}
	if ogiModule, err := NewOgImageGenerator(cmd); err == nil {
		processing = append(processing, ogiModule)
	} else {
		if !errors.Is(err, ErrModuleSkipped) {
			return err
		}
		fmt.Println("[module] ogi-image not loaded")
	}

	// File handling
	files, err := getMarkdownFiles(dir)
	if err != nil {
		return err
	}

	filesTotal := len(files)
	filesSkipped := 0
	filesPublished := 0
	filesFailed := 0
	filesMutated := 0

	for _, file := range files {
		// 0. Open and parse
		mdf, err := ParseMarkdownFile(file, dir, baseURL)
		if err != nil {
			// Failed opening or parsing
			// TODO: Handle
			continue
		}
		// TODO: mdf.IsOlderThan(period) then skip
		for _, proc := range processing {
			if err = proc.Process(ctx, mdf); err != nil {
				if verbose {
					fmt.Println(err)
				}

				if errors.Is(err, ErrFileSkipped) {
					filesSkipped++
					continue
				}
				filesFailed++
			}
		}

		for _, pub := range publishing {
			// TODO: Skip if older than pubMaxDays
			if pubMaxDays > 0 {
				cutoff := time.Now().AddDate(0, 0, -pubMaxDays)
				if mdf.Date().Before(cutoff) {
					if verbose {
						fmt.Printf("[verbose] skipped (older than %d days): %s\n", pubMaxDays, mdf.Filename)
					}
					filesSkipped++
					continue
				}
			}

			if mdf.GetSocial(pub.PublisherID()) != "" {
				if verbose {
					fmt.Println("[verbose] already published", mdf.Filename)
				}
				continue
			}
			if dryRun {
				fmt.Println("[publishing] skipped", mdf.Filename)
				continue
			}
			if err = pub.Publish(ctx, mdf); err != nil {
				if verbose {
					fmt.Println(err)
				}
				if errors.Is(err, ErrFileSkipped) {
					filesSkipped++
					continue
				}
				filesFailed++
			}
		}

		if mdf.PendingWrite && !dryRun {
			wfile, err := os.Create(file)
			if err != nil {
				return err
			}
			defer wfile.Close()
			_, err = mdf.WriteTo(wfile)
			if err != nil {
				return err
			}
			filesMutated++
		}
	}

	fmt.Printf("Found %d markdown files. %d published, %d skipped.\n", filesTotal, filesPublished, filesSkipped)

	return nil
}
