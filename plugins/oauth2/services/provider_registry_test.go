package services

import (
	"context"
	"testing"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type registryTestProvider struct {
	name string
}

func (p *registryTestProvider) Name() string                                       { return p.name }
func (p *registryTestProvider) GetConfig() *oauth2.Config                          { return &oauth2.Config{} }
func (p *registryTestProvider) GetAuthURL(string, ...oauth2.AuthCodeOption) string { return "" }
func (p *registryTestProvider) Exchange(_ context.Context, _ string, _ ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return &oauth2.Token{}, nil
}
func (p *registryTestProvider) GetUserInfo(_ context.Context, _ *oauth2.Token) (*types.UserInfo, error) {
	return &types.UserInfo{}, nil
}
func (p *registryTestProvider) RequiresPKCE() bool { return true }

func TestProviderRegistry(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()

	if err := registry.Register("google", &registryTestProvider{name: "google"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := registry.Register("Google", &registryTestProvider{name: "google"}); err == nil {
		t.Fatal("expected duplicate provider error")
	}
	if err := registry.Register("", &registryTestProvider{name: "empty"}); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := registry.Register("github", nil); err == nil {
		t.Fatal("expected nil provider error")
	}

	if err := registry.Register("discord", &registryTestProvider{name: "discord"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := registry.Names()
	if len(names) != 2 || names[0] != "discord" || names[1] != "google" {
		t.Fatalf("Names() = %v, want sorted [discord google]", names)
	}

	if _, ok := registry.Get("missing"); ok {
		t.Fatal("expected missing provider lookup to fail")
	}
}
