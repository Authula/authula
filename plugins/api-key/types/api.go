package types

import (
	"errors"
	"strings"
	"time"
)

const (
	OwnerTypeUser         = "user"
	OwnerTypeOrganization = "organization"
)

type CreateApiKeyRequest struct {
	Name                 string         `json:"name"`
	OwnerType            string         `json:"owner_type"`
	OwnerID              string         `json:"owner_id"`
	Prefix               *string        `json:"prefix,omitempty"`
	Enabled              *bool          `json:"enabled,omitempty"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty"`
	RateLimitEnabled     *bool          `json:"rate_limit_enabled,omitempty"`
	RateLimitTimeWindow  *int           `json:"rate_limit_time_window,omitempty"`
	RateLimitMaxRequests *int           `json:"rate_limit_max_requests,omitempty"`
	Permissions          []string       `json:"permissions,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

func (r *CreateApiKeyRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if r.OwnerType != OwnerTypeUser && r.OwnerType != OwnerTypeOrganization {
		return errors.New("owner_type must be either 'user' or 'organization'")
	}
	if r.OwnerID == "" {
		return errors.New("owner_id is required")
	}
	if r.RateLimitEnabled != nil && *r.RateLimitEnabled {
		if r.RateLimitTimeWindow == nil || *r.RateLimitTimeWindow <= 0 {
			return errors.New("rate_limit_time_window is required when rate limiting is enabled")
		}
		if r.RateLimitMaxRequests == nil || *r.RateLimitMaxRequests <= 0 {
			return errors.New("rate_limit_max_requests is required when rate limiting is enabled")
		}
	}
	return nil
}

type CreateApiKeyResponse struct {
	RawApiKey string  `json:"raw_api_key"`
	ApiKey    *ApiKey `json:"api_key"`
}

type GetAllApiKeysResponse struct {
	Items []*ApiKey `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

type GetApiKeysRequest struct {
	OwnerType *string
	OwnerID   *string
	Page      int
	Limit     int
}

type GetApiKeyResponse struct {
	ApiKey *ApiKey `json:"api_key"`
}

type UpdateApiKeyRequest struct {
	Name                 *string        `json:"name,omitempty"`
	Enabled              *bool          `json:"enabled,omitempty"`
	RateLimitEnabled     *bool          `json:"rate_limit_enabled,omitempty"`
	RateLimitTimeWindow  *int           `json:"rate_limit_time_window,omitempty"`
	RateLimitMaxRequests *int           `json:"rate_limit_max_requests,omitempty"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty"`
	Permissions          []string       `json:"permissions,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

func (r *UpdateApiKeyRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name cannot be empty")
	}
	if r.RateLimitEnabled != nil && *r.RateLimitEnabled {
		if r.RateLimitTimeWindow == nil || *r.RateLimitTimeWindow <= 0 {
			return errors.New("rate_limit_time_window is required when rate limiting is enabled")
		}
		if r.RateLimitMaxRequests == nil || *r.RateLimitMaxRequests <= 0 {
			return errors.New("rate_limit_max_requests is required when rate limiting is enabled")
		}
	}
	return nil
}

type UpdateApiKeyData struct {
	Name                 *string
	Enabled              *bool
	RateLimitEnabled     *bool
	RateLimitTimeWindow  *int
	RateLimitMaxRequests *int
	LastRequestedAt      *time.Time
	ExpiresAt            *time.Time
	Permissions          []string
	Metadata             map[string]any
}

type UpdateApiKeyResponse struct {
	ApiKey *ApiKey `json:"api_key"`
}

type DeleteApiKeyResponse struct {
	Message string `json:"message"`
}

type VerifyApiKeyRequest struct {
	Key string `json:"key"`
}

func (r *VerifyApiKeyRequest) Validate() error {
	if strings.TrimSpace(r.Key) == "" {
		return errors.New("key is required")
	}
	return nil
}

type VerifyApiKeyResult struct {
	Valid  bool
	ApiKey *ApiKey
}

type VerifyApiKeyResponse struct {
	ApiKey *ApiKey `json:"api_key"`
}
