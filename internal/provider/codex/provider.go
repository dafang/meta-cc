package codex

import (
	"context"
	"os"

	"github.com/yaleh/meta-cc/internal/conversation"
	"github.com/yaleh/meta-cc/internal/locator"
	"github.com/yaleh/meta-cc/internal/provider"
)

var _ provider.Provider = (*Provider)(nil)

type Provider struct {
	locator  *locator.CodexLocator
	maxLines int
}

func NewProvider(loc *locator.CodexLocator) *Provider {
	return &Provider{locator: loc, maxLines: 500_000}
}

func (p *Provider) ID() conversation.ProviderID {
	return conversation.ProviderCodex
}

func (p *Provider) IsAvailable(context.Context) bool {
	if p.locator == nil {
		return false
	}
	_, err := os.Stat(p.locator.SQLiteDB())
	return err == nil
}

func (p *Provider) ListSessions(ctx context.Context) ([]conversation.Session, error) {
	return listSessionsFromDB(ctx, p.locator.SQLiteDB())
}

func (p *Provider) GetSession(ctx context.Context, sessionID string) (conversation.Session, error) {
	return getSessionFromDB(ctx, p.locator.SQLiteDB(), sessionID)
}

func (p *Provider) LoadTurns(ctx context.Context, sessionID string) ([]conversation.Turn, error) {
	session, err := p.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return loadTurnsFromSession(session, p.maxLines)
}
