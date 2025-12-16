package services

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type RateLimitServiceImpl struct {
	config           *models.Config
	storage          models.SecondaryStorage
	logger           *slog.Logger
	pluginRateLimits []models.PluginRateLimit
}

func NewRateLimitServiceImpl(config *models.Config, pluginRateLimits []models.PluginRateLimit) *RateLimitServiceImpl {
	return &RateLimitServiceImpl{
		config:           config,
		storage:          config.SecondaryStorage.Storage,
		logger:           slog.Default(),
		pluginRateLimits: pluginRateLimits,
	}
}

// ruleFor returns the active rate limit rule for a given key/request
func (s *RateLimitServiceImpl) ruleFor(key string, req *http.Request) (time.Duration, int, bool) {
	if len(s.pluginRateLimits) > 0 {
		for _, rateLimitConfig := range s.pluginRateLimits {
			if rateLimitConfig.Enabled && rateLimitConfig.CustomRules != nil {
				if ruleFn, ok := rateLimitConfig.CustomRules[key]; ok {
					rule := ruleFn(req)
					if rule.Disabled {
						continue
					}
					return rule.Window, rule.Max, false
				}
			}
		}
	}

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
func (s *RateLimitServiceImpl) Allow(ctx context.Context, key string, req *http.Request) (bool, error) {
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
func (s *RateLimitServiceImpl) GetClientIP(req *http.Request) string {
	// Get IP from X-Forwarded-For header
	forwarded := req.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// The header can contain a comma-separated list of IPs. The first one is the original client.
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	// Get IP from X-Real-IP header
	realIP := req.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		// If splitting fails, it might be just the IP address without a port.
		return req.RemoteAddr
	}

	return ip
}

// BuildKey constructs a rate limit key for storage
func (s *RateLimitServiceImpl) BuildKey(key string) string {
	return s.config.RateLimit.Prefix + key
}
