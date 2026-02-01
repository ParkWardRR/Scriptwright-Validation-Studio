package userscript

import (
	"path/filepath"
	"testing"
)

func TestParseWikipediaScript(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "wikipedia-dark.user.js")
	meta, err := Parse(scriptPath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if meta.Name != "Wikipedia Dark/Light Mode" {
		t.Fatalf("unexpected name: %q", meta.Name)
	}
	if meta.Namespace != "http://tampermonkey.net/" {
		t.Fatalf("unexpected namespace: %q", meta.Namespace)
	}
	if len(meta.Match) == 0 || meta.Match[0] != "https://*.wikipedia.org/*" {
		t.Fatalf("unexpected match rules: %+v", meta.Match)
	}
	if meta.Version == "" {
		t.Fatal("expected version to be parsed")
	}
	if meta.Raw == "" {
		t.Fatal("expected raw content to be preserved")
	}
}
