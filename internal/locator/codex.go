package locator

import (
	"os"
	"path/filepath"
)

const codexRootEnv = "META_CC_CODEX_ROOT"

type CodexLocator struct {
	codexRoot string
}

func NewCodexLocator() *CodexLocator {
	root := os.Getenv(codexRootEnv)
	if root == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			root = filepath.Join(homeDir, ".codex")
		}
	} else {
		root = filepath.Clean(root)
	}

	return &CodexLocator{codexRoot: root}
}

func (l *CodexLocator) SQLiteDB() string {
	return filepath.Join(l.codexRoot, "state_5.sqlite")
}

func (l *CodexLocator) SessionsRoot() string {
	return filepath.Join(l.codexRoot, "sessions")
}

func (l *CodexLocator) HistoryFile() string {
	return filepath.Join(l.codexRoot, "history.jsonl")
}
