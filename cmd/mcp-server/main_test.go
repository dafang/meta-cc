package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "meta-cc-mcp-tests-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	_ = os.Setenv("CODEX_HOME", filepath.Join(tempDir, "codex-home"))
	os.Exit(m.Run())
}

func testProjectHash(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	hash := strings.ReplaceAll(resolved, "\\", "-")
	hash = strings.ReplaceAll(hash, "/", "-")
	hash = strings.ReplaceAll(hash, ":", "-")
	return hash
}
