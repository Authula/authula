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

type ApiKeyID struct {
	ID string `path:"id"`
}

type ListApiKeysQuery struct {
	OwnerType *string `query:"owner_type" json:"owner_type,omitempty" nullable:"true"`
	OwnerID   *string `query:"owner_id" json:"owner_id,omitempty" nullable:"true"`
	Page      *int    `query:"page" json:"page,omitempty" nullable:"true"`
	Limit     *int    `query:"limit" json:"limit,omitempty" nullable:"true"`
}

type CreateApiKeyRequest struct {
	Name                 string         `json:"name" required:"true" nullable:"false"`
	OwnerType            string         `json:"owner_type" required:"true" nullable:"false"`
	OwnerID              string         `json:"owner_id" required:"true" nullable:"false"`
	Prefix               *string        `json:"prefix,omitempty" nullable:"true"`
	Enabled              *bool          `json:"enabled,omitempty" nullable:"true"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty" nullable:"true"`
	RateLimitEnabled     *bool          `json:"rate_limit_enabled,omitempty" nullable:"true"`
	RateLimitTimeWindow  *int           `json:"rate_limit_time_window,omitempty" nullable:"true"`
	RateLimitMaxRequests *int           `json:"rate_limit_max_requests,omitempty" nullable:"true"`
	Permissions          []string       `json:"permissions,omitempty" nullable:"true"`
	Metadata             map[string]any `json:"metadata,omitempty" nullable:"true"`
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
	RawApiKey string  `json:"raw_api_key" required:"true" nullable:"false"`
	ApiKey    *ApiKey `json:"api_key" required:"true" nullable:"false"`
}

type GetAllApiKeysResponse struct {
	Items []*ApiKey `json:"items" required:"true" nullable:"false"`
	Total int       `json:"total" required:"true" nullable:"false"`
	Page  int       `json:"page" required:"true" nullable:"false"`
	Limit int       `json:"limit" required:"true" nullable:"false"`
}

type GetApiKeysRequest struct {
	OwnerType *string
	OwnerID   *string
	Page      int
	Limit     int
}

type GetApiKeyResponse struct {
	ApiKey *ApiKey `json:"api_key" required:"true" nullable:"false"`
}

type UpdateApiKeyRequest struct {
	Name                 *string        `json:"name,omitempty" nullable:"true"`
	Enabled              *bool          `json:"enabled,omitempty" nullable:"true"`
	RateLimitEnabled     *bool          `json:"rate_limit_enabled,omitempty" nullable:"true"`
	RateLimitTimeWindow  *int           `json:"rate_limit_time_window,omitempty" nullable:"true"`
	RateLimitMaxRequests *int           `json:"rate_limit_max_requests,omitempty" nullable:"true"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty" nullable:"true"`
	Permissions          []string       `json:"permissions,omitempty" nullable:"true"`
	Metadata             map[string]any `json:"metadata,omitempty" nullable:"true"`
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
	ApiKey *ApiKey `json:"api_key" required:"true" nullable:"false"`
}

type DeleteApiKeyResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type VerifyApiKeyRequest struct {
	Key string `json:"key" required:"true" nullable:"false"`
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
	ApiKey *ApiKey `json:"api_key" required:"true" nullable:"false"`
}
