package tests

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/uptrace/bun"
	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
)

// MockProvider is a configurable OAuth2 provider used by OAuth2 plugin tests.
type MockProvider struct {
	ProviderName string
	PKCE         bool
	UserInfo     *types.UserInfo
	UserInfoErr  error
	Token        *oauth2.Token
	ExchangeErr  error

	ExchangeCode    string
	ExchangeOpts    []oauth2.AuthCodeOption
	AuthOptions     []oauth2.AuthCodeOption
	AuthURLOverride string
}

func (p *MockProvider) Name() string { return p.ProviderName }

func (p *MockProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: "https://app.example.com/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://provider.example.com/authorize",
			TokenURL: "https://provider.example.com/token",
		},
		Scopes: []string{"email"},
	}
}

func (p *MockProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	p.AuthOptions = opts
	if p.AuthURLOverride != "" {
		return p.AuthURLOverride
	}
	return p.GetConfig().AuthCodeURL(state, opts...)
}

func (p *MockProvider) Exchange(_ context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	p.ExchangeCode = code
	p.ExchangeOpts = opts
	if p.ExchangeErr != nil {
		return nil, p.ExchangeErr
	}
	if p.Token != nil {
		return p.Token, nil
	}
	return &oauth2.Token{AccessToken: "access-token"}, nil
}

func (p *MockProvider) GetUserInfo(_ context.Context, _ *oauth2.Token) (*types.UserInfo, error) {
	if p.UserInfoErr != nil {
		return nil, p.UserInfoErr
	}
	return p.UserInfo, nil
}

func (p *MockProvider) RequiresPKCE() bool { return p.PKCE }

// NewRegistry returns a registry containing the provided providers.
func NewRegistry(providers ...types.OAuth2Provider) *services.ProviderRegistry {
	registry := services.NewProviderRegistry()
	for _, provider := range providers {
		_ = registry.Register(provider.Name(), provider)
	}
	return registry
}

// FlowStateRepositoryFake is an in-memory FlowStateRepository for use-case tests.
type FlowStateRepositoryFake struct {
	mu     sync.Mutex
	states map[string]*types.OAuthFlowState

	CreateErr  error
	GetErr     error
	ConsumeErr error
}

func NewFlowStateRepositoryFake() *FlowStateRepositoryFake {
	return &FlowStateRepositoryFake{states: make(map[string]*types.OAuthFlowState)}
}

func (f *FlowStateRepositoryFake) Create(_ context.Context, state *types.OAuthFlowState) (*types.OAuthFlowState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.CreateErr != nil {
		return nil, f.CreateErr
	}
	if _, exists := f.states[state.StateHash]; exists {
		return nil, fmt.Errorf("duplicate flow state: %s", state.StateHash)
	}
	f.states[state.StateHash] = state
	return state, nil
}

func (f *FlowStateRepositoryFake) GetByStateHash(_ context.Context, stateHash string) (*types.OAuthFlowState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return nil, f.GetErr
	}
	return f.states[stateHash], nil
}

func (f *FlowStateRepositoryFake) Consume(_ context.Context, stateHash string, provider string, purpose string, consumedAt time.Time) (*types.OAuthFlowState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ConsumeErr != nil {
		return nil, f.ConsumeErr
	}

	state, exists := f.states[stateHash]
	if !exists || state.Provider != provider || state.Purpose != purpose {
		return nil, nil
	}
	if state.ConsumedAt != nil || !state.ExpiresAt.After(consumedAt) {
		return nil, nil
	}

	consumed := consumedAt
	state.ConsumedAt = &consumed
	return state, nil
}

func (f *FlowStateRepositoryFake) DeleteExpired(_ context.Context, before time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var deleted int64
	for hash, state := range f.states {
		if state.ExpiresAt.Before(before) {
			delete(f.states, hash)
			deleted++
		}
	}
	return deleted, nil
}

func (f *FlowStateRepositoryFake) WithTx(_ bun.IDB) repositories.FlowStateRepository {
	return f
}

func (f *FlowStateRepositoryFake) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.states)
}
