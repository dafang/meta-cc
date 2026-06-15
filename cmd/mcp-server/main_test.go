package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "meta-cc-mcp-tests-*")
	if err != nil {
		panic(err)
	}

	code := m.Run()
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}
