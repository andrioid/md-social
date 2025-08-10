package main

import (
	"log"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var fmRe = regexp.MustCompile(`(?s)^(\s*?)---\s*\n(.*?)\n---\s*\n?`)

func parseMDContents(inputData []byte) Parsed {
	// Drop BOM
	input := string(inputData)
	input = strings.TrimPrefix(input, "\uFEFF")
	m := fmRe.FindStringSubmatchIndex(input)
	if m == nil || m[0] != 0 {
		return Parsed{Frontmatter: map[string]FMValue{}, Body: input, HasFrontmatter: false, RawBlock: ""}
	}
	raw := input[m[4]:m[5]]
	body := input[m[1*2]:] // full match length
	data, err := parseFrontMatter([]byte(raw))
	if err != nil {
		log.Fatal(err)
	}
	return Parsed{Frontmatter: data, Body: body, HasFrontmatter: true, RawBlock: raw}
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
