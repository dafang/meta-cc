package provider

import (
	"context"
	"log/slog"

	"github.com/yaleh/meta-cc/internal/conversation"
)

type Registry struct {
	providers map[conversation.ProviderID]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	reg := &Registry{providers: make(map[conversation.ProviderID]Provider, len(providers))}
	for _, p := range providers {
		if p != nil {
			reg.providers[p.ID()] = p
		}
	}
	return reg
}

func (r *Registry) Providers(ids []conversation.ProviderID) []Provider {
	if len(ids) == 0 {
		out := make([]Provider, 0, len(r.providers))
		for _, p := range r.providers {
			out = append(out, p)
		}
		return out
	}

	out := make([]Provider, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.providers[id]; ok {
			out = append(out, p)
		}
	}
	return out
}

func (r *Registry) MergedSessions(ctx context.Context, providerFilter []conversation.ProviderID) ([]conversation.Session, error) {
	var merged []conversation.Session
	for _, p := range r.Providers(providerFilter) {
		if !p.IsAvailable(ctx) {
			slog.Warn("provider unavailable", "provider", p.ID())
			continue
		}
		sessions, err := p.ListSessions(ctx)
		if err != nil {
			return nil, err
		}
		merged = append(merged, sessions...)
	}
	return merged, nil
}
