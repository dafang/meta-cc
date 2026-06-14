package provider

import (
	"context"

	"github.com/yaleh/meta-cc/internal/conversation"
)

type Provider interface {
	ID() conversation.ProviderID
	IsAvailable(ctx context.Context) bool
	ListSessions(ctx context.Context) ([]conversation.Session, error)
	GetSession(ctx context.Context, sessionID string) (conversation.Session, error)
	LoadTurns(ctx context.Context, sessionID string) ([]conversation.Turn, error)
}
