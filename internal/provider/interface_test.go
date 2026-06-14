package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yaleh/meta-cc/internal/conversation"
)

type mockProvider struct {
	id        conversation.ProviderID
	available bool
	sessions  []conversation.Session
	err       error
}

func (m *mockProvider) ID() conversation.ProviderID      { return m.id }
func (m *mockProvider) IsAvailable(context.Context) bool { return m.available }
func (m *mockProvider) ListSessions(context.Context) ([]conversation.Session, error) {
	return m.sessions, m.err
}
func (m *mockProvider) GetSession(context.Context, string) (conversation.Session, error) {
	return conversation.Session{}, errors.New("unused")
}
func (m *mockProvider) LoadTurns(context.Context, string) ([]conversation.Turn, error) {
	return nil, errors.New("unused")
}

func TestRegistryMergedSessions(t *testing.T) {
	sess := conversation.Session{ID: "s1", Provider: conversation.ProviderClaude, CWD: "/tmp", CreatedAt: time.Unix(1, 0).UTC()}
	reg := NewRegistry(
		&mockProvider{id: conversation.ProviderClaude, available: true, sessions: []conversation.Session{sess}},
		&mockProvider{id: conversation.ProviderCodex, available: false},
	)

	got, err := reg.MergedSessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("MergedSessions: %v", err)
	}
	if len(got) != 1 || got[0].ID != "s1" {
		t.Fatalf("unexpected merged sessions: %#v", got)
	}
}

func TestRegistryFilter(t *testing.T) {
	reg := NewRegistry(
		&mockProvider{id: conversation.ProviderClaude, available: true},
		&mockProvider{id: conversation.ProviderCodex, available: true, sessions: []conversation.Session{{ID: "c1", Provider: conversation.ProviderCodex, CWD: "/tmp", CreatedAt: time.Unix(1, 0).UTC()}}},
	)

	got, err := reg.MergedSessions(context.Background(), []conversation.ProviderID{conversation.ProviderCodex})
	if err != nil {
		t.Fatalf("MergedSessions: %v", err)
	}
	if len(got) != 1 || got[0].Provider != conversation.ProviderCodex {
		t.Fatalf("unexpected filtered sessions: %#v", got)
	}
}
