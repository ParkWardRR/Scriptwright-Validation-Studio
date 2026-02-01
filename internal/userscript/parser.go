package userscript

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// Meta holds the parsed metadata block of a userscript.
type Meta struct {
	Name        string
	Namespace   string
	Version     string
	Description string
	Match       []string
	Include     []string
	Exclude     []string
	RunAt       string
	Grants      []string
	Raw         string
}

// Parse reads a userscript from path and returns its metadata.
// It expects a standard // ==UserScript== header block.
func Parse(path string) (Meta, error) {
	file, err := os.Open(path)
	if err != nil {
		return Meta{}, err
	}
	defer file.Close()

	var (
		meta Meta
		in   bool
		raw  strings.Builder
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		raw.WriteString(line + "\n")

		if strings.HasPrefix(line, "// ==UserScript==") {
			in = true
			continue
		}
		if strings.HasPrefix(line, "// ==/UserScript==") {
			in = false
			break
		}
		if !in {
			continue
		}
		parts := strings.Fields(strings.TrimPrefix(line, "//"))
		if len(parts) < 2 || !strings.HasPrefix(parts[0], "@") {
			continue
		}
		key := strings.TrimPrefix(parts[0], "@")
		val := strings.TrimSpace(strings.TrimPrefix(line, "// "+parts[0]))
		switch key {
		case "name":
			meta.Name = val
		case "namespace":
			meta.Namespace = val
		case "version":
			meta.Version = val
		case "description":
			meta.Description = val
		case "match":
			meta.Match = append(meta.Match, val)
		case "include":
			meta.Include = append(meta.Include, val)
		case "exclude":
			meta.Exclude = append(meta.Exclude, val)
		case "run-at":
			meta.RunAt = val
		case "grant":
			meta.Grants = append(meta.Grants, val)
		}
	}
	if err := scanner.Err(); err != nil {
		return Meta{}, err
	}
	meta.Raw = raw.String()

	if meta.Name == "" {
		return Meta{}, errors.New("missing @name in userscript metadata")
	}
	return meta, nil
}
