package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaleh/meta-cc/internal/conversation"
	"github.com/yaleh/meta-cc/internal/locator"
)

func TestProviderID(t *testing.T) {
	p := NewProvider(locator.NewSessionLocator(), ".")
	if got := p.ID(); got != conversation.ProviderClaude {
		t.Fatalf("ID() = %s", got)
	}
}

func TestIsAvailable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", root)
	if !NewProvider(locator.NewSessionLocator(), ".").IsAvailable(context.Background()) {
		t.Fatalf("expected available")
	}

	t.Setenv("META_CC_PROJECTS_ROOT", filepath.Join(root, "missing"))
	if NewProvider(locator.NewSessionLocator(), ".").IsAvailable(context.Background()) {
		t.Fatalf("expected unavailable")
	}
}

func TestListSessionsAndLoadTurns(t *testing.T) {
	root := t.TempDir()
	project := t.TempDir()
	resolvedProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, strings.NewReplacer("\\", "-", "/", "-", ":", "-").Replace(resolvedProject))
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "sample-session.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sessionFile := filepath.Join(projectDir, "sample.jsonl")
	if err := os.WriteFile(sessionFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("META_CC_PROJECTS_ROOT", root)
	p := NewProvider(locator.NewSessionLocator(), project)
	sessions, err := p.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Provider != conversation.ProviderClaude {
		t.Fatalf("unexpected sessions: %#v", sessions)
	}

	turns, err := p.LoadTurns(context.Background(), sessions[0].ID)
	if err != nil {
		t.Fatalf("LoadTurns: %v", err)
	}
	if len(turns) != 2 && len(turns) != 1 {
		t.Fatalf("unexpected turns: %#v", turns)
	}
	if len(turns) > 0 && len(turns[0].ToolCalls) > 0 && turns[0].ToolCalls[0].Name != "Grep" {
		t.Fatalf("unexpected tool call: %#v", turns[0].ToolCalls[0])
	}
}
