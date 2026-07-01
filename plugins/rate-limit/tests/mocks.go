package tests

import (
	"context"
	"time"

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
	delete(p.Store, key)
	return nil
}

func (p *FakeRateLimitProvider) Close() error { return nil }
