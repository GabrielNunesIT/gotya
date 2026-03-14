package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoader_CircularImport(t *testing.T) {
	dir := t.TempDir()

	modA := `module mod-a {
    namespace "urn:mod-a";
    prefix "a";
    import mod-b { prefix "b"; }
}`
	modB := `module mod-b {
    namespace "urn:mod-b";
    prefix "b";
    import mod-a { prefix "a"; }
}`

	err := os.WriteFile(filepath.Join(dir, "mod-a.yang"), []byte(modA), 0600)
	if err != nil {
		t.Fatalf("write mod-a.yang: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, "mod-b.yang"), []byte(modB), 0600)
	if err != nil {
		t.Fatalf("write mod-b.yang: %v", err)
	}

	loader := NewDirectoryLoader([]string{dir})
	_, loadErr := loader.Load("mod-a")

	assert.Error(t, loadErr, "expected error for circular import")
	assert.True(t, strings.Contains(loadErr.Error(), "circular import"),
		"expected error to contain 'circular import', got: %v", loadErr)
}
