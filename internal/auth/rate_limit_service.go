package auth

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type RateLimitService struct {
	config  *domain.Config
	storage domain.SecondaryStorage
	logger  *slog.Logger
}

func NewRateLimitService(config *domain.Config) *RateLimitService {
	return &RateLimitService{
		config:  config,
		storage: config.SecondaryStorage.Storage,
		logger:  slog.Default(),
	}
}

// ruleFor returns the active rate limit rule for a given key/request
func (s *RateLimitService) ruleFor(key string, req *http.Request) (time.Duration, int, bool) {
	if ruleFn, ok := s.config.RateLimit.CustomRules[key]; ok {
		rule := ruleFn(req)
		if rule.Disabled {
			return 0, 0, true
		}
		return rule.Window, rule.Max, false
	}

	return s.config.RateLimit.Window, s.config.RateLimit.Max, false
}

// Allow checks if a request is allowed based on rate limiting rules
func (s *RateLimitService) Allow(ctx context.Context, key string, req *http.Request) (bool, error) {
	if !s.config.RateLimit.Enabled {
		return true, nil
	}

	window, max, disabled := s.ruleFor(req.URL.Path, req)
	if disabled {
		return true, nil
	}

	var count int
	value, err := s.storage.Get(ctx, key)
	if err == nil && value != nil {
		switch v := value.(type) {
		case string:
			if num, err := strconv.Atoi(v); err == nil {
				count = num
			}
		case int:
			count = v
		}
	}

	if count >= max {
		s.logger.Debug("rate limit exceeded", slog.String("key", key), slog.Int("count", count), slog.Int("max", max))
		return false, nil
	}

	ttl := window
	if _, err := s.storage.Incr(ctx, key, &ttl); err != nil {
		s.logger.Error("rate limit storage incr error", slog.String("key", key), slog.Any("error", err))
		return false, err
	}

	return true, nil
}

// GetClientIP extracts the client's IP address from the request based on configured headers
func (s *RateLimitService) GetClientIP(req *http.Request) string {
	for _, h := range s.config.RateLimit.IP.Headers {
		if val := req.Header.Get(h); val != "" {
			// X-Forwarded-For may contain comma-separated list
			parts := strings.Split(val, ",")
			ip := strings.TrimSpace(parts[0])
			// Strip port if present
			if host, _, err := net.SplitHostPort(ip); err == nil {
				return host
			}
			return ip
		}
	}

	// fallback to RemoteAddr
	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		return host
	}

	return req.RemoteAddr
}
