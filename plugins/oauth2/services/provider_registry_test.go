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

func TestValidateProviderName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "simple", input: "google"},
		{name: "digits", input: "provider1"},
		{name: "hyphen and underscore", input: "my-provider_1"},
		{name: "empty", input: "", wantErr: true},
		{name: "path traversal", input: "../admin", wantErr: true},
		{name: "slash", input: "a/b", wantErr: true},
		{name: "uppercase", input: "Google", wantErr: true},
		{name: "leading hyphen", input: "-google", wantErr: true},
		{name: "too long", input: string(make([]byte, 65)), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateProviderName(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error for %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}

func TestProviderRegistry(t *testing.T) {
	t.Parallel()

	registerSteps := []struct {
		name     string
		provider types.OAuth2Provider
		wantErr  bool
	}{
		{name: "google", provider: &registryTestProvider{name: "google"}},
		{name: "Google", provider: &registryTestProvider{name: "google"}, wantErr: true},
		{name: "", provider: &registryTestProvider{name: "empty"}, wantErr: true},
		{name: "github", provider: nil, wantErr: true},
		{name: "discord", provider: &registryTestProvider{name: "discord"}},
	}

	registry := NewProviderRegistry()
	for _, step := range registerSteps {
		t.Run("register_"+step.name, func(t *testing.T) {
			err := registry.Register(step.name, step.provider)
			if step.wantErr && err == nil {
				t.Fatalf("expected an error registering %q", step.name)
			}
			if !step.wantErr && err != nil {
				t.Fatalf("unexpected error registering %q: %v", step.name, err)
			}
		})
	}

	lookupTests := []struct {
		name      string
		wantNames []string
		lookup    string
		wantFound bool
	}{
		{name: "names are sorted and canonical", wantNames: []string{"discord", "google"}, lookup: "missing", wantFound: false},
		{name: "registered provider is found", wantNames: []string{"discord", "google"}, lookup: "google", wantFound: true},
	}

	for _, tt := range lookupTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registry.Names(); len(got) != len(tt.wantNames) || got[0] != tt.wantNames[0] || got[1] != tt.wantNames[1] {
				t.Fatalf("Names() = %v, want %v", got, tt.wantNames)
			}
			if _, ok := registry.Get(tt.lookup); ok != tt.wantFound {
				t.Fatalf("Get(%q) found = %v, want %v", tt.lookup, ok, tt.wantFound)
			}
		})
	}
}
