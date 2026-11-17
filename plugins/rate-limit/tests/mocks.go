package tests

import (
	"context"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/rate-limit/types"
)

type FakeRateLimitProvider struct {
	CheckAllowed bool
	CheckCount   int
	CheckReset   time.Time
	CheckErr     error

	GetErr error
	SetErr error
	Store  map[string]types.RateLimitRuleRecord

	GetValueErr   error
	DeleteRuleErr error

	LastCheckKey    string
	LastCheckWindow time.Duration
	LastCheckMax    int
	GetCalls        int
	SetCalls        int
	LastSetKey      string
	LastSetWindow   time.Duration
	LastSetMax      int
}

func NewFakeRateLimitProvider() *FakeRateLimitProvider {
	return &FakeRateLimitProvider{Store: make(map[string]types.RateLimitRuleRecord)}
}

func (p *FakeRateLimitProvider) GetName() string { return "fake" }

func (p *FakeRateLimitProvider) GetValue(_ context.Context, key string) (any, error) {
	if p.GetValueErr != nil {
		return nil, p.GetValueErr
	}
	record, ok := p.Store[key]
	if !ok {
		return nil, nil
	}
	return types.RateLimitValue{
		Count:     record.MaxRequests,
		ExpiresAt: time.Unix(int64(record.WindowSeconds), 0),
	}, nil
}

func (p *FakeRateLimitProvider) WithCheckResult(allowed bool, count int, reset time.Time, err error) *FakeRateLimitProvider {
	p.CheckAllowed = allowed
	p.CheckCount = count
	p.CheckReset = reset
	p.CheckErr = err
	return p
}

func (p *FakeRateLimitProvider) WithCheckError(err error) *FakeRateLimitProvider {
	p.CheckErr = err
	return p
}

func (p *FakeRateLimitProvider) WithExistingRule(key string, window time.Duration, max int) *FakeRateLimitProvider {
	p.Store[key] = types.RateLimitRuleRecord{Key: key, WindowSeconds: int(window.Seconds()), MaxRequests: max}
	return p
}

func (p *FakeRateLimitProvider) WithGetError(err error) *FakeRateLimitProvider {
	p.GetErr = err
	return p
}

func (p *FakeRateLimitProvider) WithSetError(err error) *FakeRateLimitProvider {
	p.SetErr = err
	return p
}

func (p *FakeRateLimitProvider) CheckAndIncrement(_ context.Context, key string, window time.Duration, maxRequests int) (bool, int, time.Time, error) {
	p.LastCheckKey = key
	p.LastCheckWindow = window
	p.LastCheckMax = maxRequests
	if p.CheckReset.IsZero() {
		p.CheckReset = time.Unix(1000, 0)
	}
	return p.CheckAllowed, p.CheckCount, p.CheckReset, p.CheckErr
}

func (p *FakeRateLimitProvider) SetRule(_ context.Context, key string, window time.Duration, maxRequests int) error {
	p.SetCalls++
	p.LastSetKey = key
	p.LastSetWindow = window
	p.LastSetMax = maxRequests
	if p.SetErr != nil {
		return p.SetErr
	}
	p.Store[key] = types.RateLimitRuleRecord{Key: key, WindowSeconds: int(window.Seconds()), MaxRequests: maxRequests}
	return nil
}

func (p *FakeRateLimitProvider) GetRule(_ context.Context, key string) (time.Duration, int, bool, error) {
	p.GetCalls++
	if p.GetErr != nil {
		return 0, 0, false, p.GetErr
	}
	record, ok := p.Store[key]
	if !ok {
		return 0, 0, false, nil
	}
	return time.Duration(record.WindowSeconds) * time.Second, record.MaxRequests, true, nil
}

func (p *FakeRateLimitProvider) DeleteRule(_ context.Context, key string) error {
	if p.DeleteRuleErr != nil {
		return p.DeleteRuleErr
	}
	delete(p.Store, key)
	return nil
}

func (p *FakeRateLimitProvider) Close() error { return nil }

// StubSecondaryStorageService is a minimal SecondaryStorageService for plugin tests.
type StubSecondaryStorageService struct {
	Storage models.SecondaryStorage
	Name    string
}

func (s *StubSecondaryStorageService) GetStorage() models.SecondaryStorage { return s.Storage }

func (s *StubSecondaryStorageService) GetProviderName() string { return s.Name }

// RecordingServiceRegistry records registrations so tests can assert wiring.
type RecordingServiceRegistry struct {
	Services   map[string]any
	Registered []string
}

func (r *RecordingServiceRegistry) Register(name string, service any) {
	if r.Services == nil {
		r.Services = make(map[string]any)
	}
	r.Services[name] = service
	r.Registered = append(r.Registered, name)
}

func (r *RecordingServiceRegistry) Get(name string) any {
	if r == nil || r.Services == nil {
		return nil
	}
	return r.Services[name]
}
