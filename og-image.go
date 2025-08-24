package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ogImageGenerator struct {
	t         *template.Template
	overwrite bool
	args      templateArguments
	// We disable the processor if no resvg is found
	disabled bool
}

type templateArguments struct {
	Title           string
	SubTitle        string
	BackgroundImage template.URL
	BackgroundWidth int
	// height is `width / 1.91`
	BackgroundHeight int
	AuthorImage      string
	AuthorWidth      template.URL
	AuthorHeight     int
}

//go:embed *.svg
var templateFS embed.FS

func NewOgImageGenerator(overwrite bool) (*ogImageGenerator, error) {
	disabled := false
	if _, err := exec.LookPath("resvg"); err != nil {
		fmt.Println("warning: resvg was not found, skipping og-images")
		disabled = true
	}
	if ogImageBackground == "" {
		fmt.Println("OG_IMAGE_BG not defined, skipping og-images")
		disabled = true
	}
	//Compile template
	tmpl, err := template.ParseFS(templateFS, "og-image-template.svg")
	if err != nil {
		return nil, err
	}
	//Translate assets into base64 and cache
	return &ogImageGenerator{
		t:         tmpl,
		overwrite: overwrite,
		args: templateArguments{
			Title:    "demotitle",
			SubTitle: "demosubtitle",
		},
		disabled: disabled,
	}, nil
}

// TODO: Refactor the publish API to use processer interface instead
func (ogi *ogImageGenerator) Process(ctx context.Context, mdf *MDFile) error {
	if ogi.disabled {
		return nil
	}
	fmt.Println("ogi process called")
	ext := filepath.Ext(mdf.Filename)
	basename, found := strings.CutSuffix(mdf.Filename, ext)
	if !found {
		return fmt.Errorf("md file didnt have extension")
	}
	svgFn := filepath.Join(mdf.BaseDir, basename) + ".svg"
	pngFn := filepath.Join(mdf.BaseDir, basename) + ".png"
	fmt.Println("basename", svgFn)
	svgFile, err := os.Create(svgFn)
	if err != nil {
		return err
	}
	defer svgFile.Close()

	img64, err := FileToDataURL(ogImageBackground)
	if err != nil {
		return err
	}

	p := mdf.GetPost()
	err = ogi.t.Execute(svgFile, templateArguments{
		Title:           p.title,
		SubTitle:        p.date.Format("2006-01-02"),
		BackgroundImage: template.URL(img64),
	})

	if err != nil {
		return err
	}

	// Convert to png
	cmd := exec.Command("resvg", svgFn, pngFn)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert svg to png with resvg: %w", err)
	}

	mdf.FM["ogImage"] = pngFn
	mdf.PendingWrite = true

	return nil
}
