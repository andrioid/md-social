package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
)

type ogImageGenerator struct {
	t         *template.Template
	overwrite bool
	args      templateArguments
	cmd       *cli.Command
	destDir   string
	// We disable the processor if no resvg is found
}

type templateArguments struct {
	Title           string
	SubTitle        string
	BackgroundImage template.URL
	BackgroundWidth int
	// height is `width / 1.91`
	BackgroundHeight int
	AuthorImage      template.URL
	AuthorWidth      int
	AuthorHeight     int
}

func NewOgImageGenerator(cmd *cli.Command) (*ogImageGenerator, error) {
	bgImagePath := cmd.String("og-image-bg")
	bgImage := ""
	authorImagePath := cmd.String("og-image-author")
	authorImage := ""

	assetFS, err := fs.Sub(embedFS, "assets")
	if err != nil {
		return nil, err
	}

	if _, err := exec.LookPath("resvg"); err != nil {
		fmt.Println("warning: resvg was not found, skipping og-images")
		return nil, err
	}
	// TODO: Handle default background image

	//Compile template
	tmpl, err := template.ParseFS(assetFS, "og-image-template.svg")
	if err != nil {
		return nil, err
	}
	//Translate assets into base64 and cache
	if bgImagePath != "" {
		bgImage, err = FileToDataURL(bgImagePath)
		if err != nil {
			log.Fatal(err)
		}
	}
	if authorImagePath != "" {
		authorImage, err = FileToDataURL(authorImagePath)
		if err != nil {
			log.Fatal(err)
		}
	}

	ogDir := cmd.String("og-dest-dir")
	if ogDir == "" {
		ogDir = cmd.StringArg("dir")
	}

	ogi := &ogImageGenerator{
		t:         tmpl,
		overwrite: cmd.Bool("og-image-overwrite"),
		args: templateArguments{
			// Placeholder values
			Title:           "",
			SubTitle:        "",
			BackgroundImage: template.URL(bgImage),
			AuthorImage:     template.URL(authorImage),
		},
		destDir: ogDir,
		cmd:     cmd,
	}
	return ogi, nil
}

func (ogi *ogImageGenerator) Process(ctx context.Context, mdf *MDFile) error {
	ext := filepath.Ext(mdf.Filename)
	basename, found := strings.CutSuffix(mdf.Filename, ext)
	if !found {
		return fmt.Errorf("md file didnt have extension")
	}
	basename = strings.TrimPrefix(basename, "/")

	pngFn := filepath.Join(ogi.destDir, basename) + ".png"
	svgFn := filepath.Join(ogi.destDir, basename) + ".svg"

	// Create destination directory, if it doesn't exist
	dstDir := filepath.Dir(pngFn)
	if _, err := os.Stat(dstDir); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(dstDir, 0755)
		if err != nil {
			return err
		}
	}

	//fmt.Println("basename", svgFn)
	svgFile, err := os.Create(svgFn)
	if err != nil {
		return err
	}
	defer svgFile.Close()
	if !ogi.cmd.Bool("og-keep-svg") {
		defer os.Remove(svgFile.Name())
	}

	templateArgs := ogi.args
	templateArgs.Title = mdf.Title()
	templateArgs.SubTitle = mdf.Description()
	if mdf.CoverImage() != "" {
		bgp := filepath.Join(mdf.BaseDir, filepath.Dir(mdf.Filename), mdf.CoverImage())
		b64, err := FileToDataURL(bgp)
		if err != nil {
			return err
		}
		templateArgs.BackgroundImage = template.URL(b64)
	}
	// TODO: Allow image overrides too

	err = ogi.t.Execute(svgFile, templateArgs)

	if err != nil {
		return err
	}

	// Convert to png
	cmd := exec.Command("resvg", "--font-family", "sans-serif", svgFile.Name(), pngFn)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert svg to png with resvg: %w", err)
	}

	ogUrl := basename + ".png"
	mdf.FM["ogImage"] = ogUrl
	mdf.PendingWrite = true

	return nil
}
