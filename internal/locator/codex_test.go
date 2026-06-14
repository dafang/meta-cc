package locator

import (
	"path/filepath"
	"testing"
)

func TestCodexLocatorEnvOverride(t *testing.T) {
	t.Setenv(codexRootEnv, "/tmp/codex")
	loc := NewCodexLocator()
	if got := loc.SQLiteDB(); got != "/tmp/codex/state_5.sqlite" {
		t.Fatalf("SQLiteDB() = %s", got)
	}
}

func TestCodexLocatorDefaultPaths(t *testing.T) {
	t.Setenv(codexRootEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	loc := NewCodexLocator()

	if got := loc.SessionsRoot(); got != filepath.Join(home, ".codex", "sessions") {
		t.Fatalf("SessionsRoot() = %s", got)
	}
	if got := loc.HistoryFile(); got != filepath.Join(home, ".codex", "history.jsonl") {
		t.Fatalf("HistoryFile() = %s", got)
	}
}
