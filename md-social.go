package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

type FMValue = any

var (
	ErrInvalidFile   = errors.New("invalid markdown file")
	ErrFileSkipped   = errors.New("file skipped")
	ErrModuleSkipped = errors.New("module skipped")
)

// Write more debug info
var verbose = false

// Don't persist any file changes, or post anything online
var dryRun = false

func main() {
	cmd := &cli.Command{
		Usage: "Parse markdown articles, do stuff with them",
		Commands: []*cli.Command{
			{
				Name:      "parse",
				Usage:     "Parses a directory for frontmatter markdown files.",
				ArgsUsage: "blogposts/",
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
					&cli.BoolFlag{
						Name:    "og-image-overwrite",
						Usage:   "Overwrite existing og-images",
						Value:   true,
						Sources: cli.EnvVars("OG_OVERWRITE"),
					},
					&cli.StringFlag{
						Name:    "og-dest-dir",
						Usage:   "destination directory for og-images. If empty, it will write to same directory as input",
						Sources: cli.EnvVars("OG_DEST_DIR"),
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
				Sources: cli.EnvVars("DRYRUN"),
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Value:   false,
				Usage:   "More verbose output for debugging purposes.",
				Sources: cli.EnvVars("VERBOSE"),
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
			&cli.IntFlag{
				Name:    "publish-max-days",
				Usage:   "Don't publish anything older than this",
				Sources: cli.EnvVars("PUBLISH_MAX_DAYS"),
			},
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println("[application error]", err)
		os.Exit(1)
	}
}
