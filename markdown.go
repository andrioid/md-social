package main

import (
	"io"
	"log"
	"net/url"
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
}

var fmRe = regexp.MustCompile(`(?s)^(\s*?)---\s*\n(.*?)\n---\s*\n?`)

func Parse(inputData []byte, fpath, prefix string) *MDFile {
	// Drop BOM
	input := string(inputData)
	body := strings.TrimPrefix(input, "\uFEFF")
	m := fmRe.FindStringSubmatchIndex(input)

	filename := ""
	if relativePath, hasPrefix := strings.CutPrefix(fpath, prefix); hasPrefix {
		filename = relativePath
	} else {
		filename = fpath
	}

	// No frontmatter, no point
	if m == nil || m[0] != 0 {
		return nil
	}

	rawFM := input[m[4]:m[5]]
	body = input[m[1]:] // full match length
	fm, err := parseFrontMatter([]byte(rawFM))
	if err != nil {
		log.Fatal(err)
	}
	return &MDFile{
		FM:           fm,
		Filename:     filename,
		Body:         body,
		RawFM:        rawFM,
		PendingWrite: false,
	}
}

var kvRe = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:\s*(.*)$`)

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

type PostDetails struct {
	title string
	url   string
	date  time.Time
}

func (mdf *MDFile) GetPost() PostDetails {
	p := PostDetails{}

	title, ok := mdf.FM["title"].(string)
	if ok {
		p.title = title
	}

	slug, ok := mdf.FM["slug"].(string)
	if ok {
		p.url, _ = url.JoinPath(baseURL, slug)
	} else {
		// If no slug, use filename
		p.url, _ = url.JoinPath(baseURL, mdf.Filename)
	}

	datestr, ok := mdf.FM["date"].(string)
	if ok {
		d, err := time.Parse(time.RFC3339, datestr)
		if err == nil {
			p.date = d
		}
	}

	return p
}
