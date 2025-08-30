package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"
)

type FMValue = any

var (
	ErrInvalidFile = errors.New("invalid markdown file")
	ErrSkipped     = errors.New("file skipped")
)

var publishers []Publisher

func main() {
	cmd := &cli.Command{
		Usage: "Parse markdown articles, do stuff with them",
		Commands: []*cli.Command{
			{
				Name:      "parse",
				Usage:     "Parses a directory for frontmatter markdown files.",
				ArgsUsage: "parse blogposts/",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "dir",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "og-image",
						Value:   false,
						Usage:   "create og-image for markdown files",
						Sources: cli.EnvVars("OG_IMAGE"),
					},
					&cli.StringFlag{
						Name:    "og-image-bg",
						Usage:   "image file to use as background",
						Value:   "",
						Sources: cli.EnvVars("OG_IMAGE_BG"),
					},
					&cli.StringFlag{
						Name:    "og-image-author",
						Usage:   "image file to use as author image",
						Value:   "",
						Sources: cli.EnvVars("OG_IMAGE_AUTHOR"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return parse(ctx, cmd)
				},
			},
			{
				Name:  "server",
				Usage: "Server to help with services that require OAUTH",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "base-url",
				Usage:    "Used to create links back to the article",
				Required: true,
				Sources:  cli.EnvVars("BASE_URL"),
			},

			&cli.BoolFlag{
				Name:    "dryrun",
				Value:   false,
				Usage:   "Dry run. Doesn't post, or write to disk",
				Sources: cli.EnvVars("DRY_RUN"),
			},
			&cli.StringFlag{
				Name:    "bluesky-handle",
				Usage:   "Bluesky handle to use when posting. If not set, Bluesky integration is disabled.",
				Sources: cli.EnvVars("BLUESKY_HANDLE"),
			},
			&cli.StringFlag{
				Name:    "bluesky-app-pw",
				Usage:   "Bluesky app password to use when posting",
				Sources: cli.EnvVars("BLUESKY_APP_PASSWORD"),
			},
		},
	}
	cmd.Run(context.Background(), os.Args)
}

func parse(ctx context.Context, cmd *cli.Command) error {
	dir := cmd.Args().First()
	bskyHandle := cmd.String("bluesky-handle")
	bskyAppPW := cmd.String("bluesky-app-pw")
	baseURL := cmd.String("base-url")

	publishers = []Publisher{}
	// Publishers
	if bskyHandle != "" {
		publishers = []Publisher{NewBluesky(ctx)}
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
