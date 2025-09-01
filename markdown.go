package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type MDFile struct {
	// Frontmatter
	FM map[string]any
	// Markdown body
	Body string
	// Frontmatter raw
	RawFM        string // without --- markers
	PendingWrite bool   // If we need to write it or not
	// Relative filename, used for URL if no slug is specified
	Filename string
	BaseDir  string
	BaseURL  string
}

var fmRe = regexp.MustCompile(`(?s)^(\s*?)---\s*\n(.*?)\n---\s*\n?`)

func ParseMarkdownFile(file, prefix, baseURL string) (*MDFile, error) {
	inputData, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// Drop BOM
	input := string(inputData)
	body := strings.TrimPrefix(input, "\uFEFF")
	m := fmRe.FindStringSubmatchIndex(input)

	if relativePath, hasPrefix := strings.CutPrefix(file, prefix); hasPrefix {
		file = relativePath
	}

	// No frontmatter, no point
	if m == nil || m[0] != 0 {
		return nil, fmt.Errorf("%w: no frontmatter", ErrFileSkipped)
	}

	rawFM := input[m[4]:m[5]]
	body = input[m[1]:] // full match length
	fm, err := parseFrontMatter([]byte(rawFM))
	if err != nil {
		log.Fatal(err)
	}

	mdf := &MDFile{
		FM:           fm,
		Filename:     file,
		BaseDir:      prefix,
		Body:         body,
		RawFM:        rawFM,
		PendingWrite: false,
		BaseURL:      baseURL,
	}

	if mdf.Title() == "" || mdf.URL() == "" {
		return nil, fmt.Errorf("%w: no title or url in frontmatter: %s", ErrFileSkipped, file)
	}
	if mdf.Date().IsZero() {
		return nil, fmt.Errorf("%w: date not found or empty: %s", ErrFileSkipped, file)
	}
	return mdf, nil
}

func parseFrontMatter(data []byte) (map[string]any, error) {
	d := map[string]any{}
	err := yaml.Unmarshal(data, &d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (mdf *MDFile) WriteTo(w io.Writer) (int64, error) {
	var content strings.Builder

	content.WriteString("---\n")

	yamlData, err := yaml.Marshal(mdf.FM)
	if err != nil {
		return 0, err
	}
	content.Write(yamlData)
	content.WriteString("---\n")

	content.WriteString(mdf.Body)

	n, err := io.WriteString(w, content.String())
	if err == nil {
		mdf.PendingWrite = false
	}

	return int64(n), err
}

func (mdf *MDFile) SetSocial(key, url string) {
	// Existing
	social, ok := mdf.FM["social"].(map[string]string)
	if !ok {
		social = map[string]string{}
	}

	social[key] = url
	mdf.FM["social"] = social
	mdf.PendingWrite = true
}

func (mdf *MDFile) GetSocial(key string) string {
	social, ok := mdf.FM["social"].(map[string]any)
	if !ok {
		return ""
	}
	val, ok := social[key].(string)
	if !ok {
		return ""
	}
	return val
}

func (mdf *MDFile) URL() string {
	slug, ok := mdf.FM["slug"].(string)
	if ok {
		purl, _ := url.JoinPath(mdf.BaseURL, slug)
		return purl
	}
	// If no slug, use filename, without extension
	name := mdf.Filename
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)
	purl, _ := url.JoinPath(mdf.BaseURL, name)
	return purl
}

func (mdf *MDFile) Date() time.Time {
	datestr, ok := mdf.FM["date"].(string)
	if ok {
		d, err := time.Parse(time.RFC3339, datestr)
		if err == nil {
			return d
		}
	}
	return time.Time{}
}

func (mdf *MDFile) Title() string {
	title, ok := mdf.FM["title"].(string)
	if ok {
		return title
	}
	return ""

}

func (mdf *MDFile) CoverImage() string {
	if val, ok := mdf.FM["coverImage"].(string); ok {
		return val
	}
	return ""
}

func (mdf *MDFile) Tags() []string {
	if val, ok := mdf.FM["coverImage"].([]string); ok {
		return val
	}
	return []string{}

}
