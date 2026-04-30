package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	OwnerTypeUser         = "user"
	OwnerTypeOrganization = "organization"
)

type CreateApiKeyRequest struct {
	Name                 string          `json:"name" validate:"required"`
	OwnerType            string          `json:"owner_type" validate:"required,oneof=user organization"`
	ReferenceID          string          `json:"reference_id" validate:"required"`
	Prefix               *string         `json:"prefix,omitempty"`
	Enabled              *bool           `json:"enabled,omitempty"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
	RateLimitEnabled     *bool           `json:"rate_limit_enabled,omitempty"`
	RateLimitTimeWindow  *int            `json:"rate_limit_time_window,omitempty"`
	RateLimitMaxRequests *int            `json:"rate_limit_max_requests,omitempty"`
	RequestsRemaining    *int            `json:"requests_remaining,omitempty"`
	Permissions          []string        `json:"permissions,omitempty"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
}

func (r *CreateApiKeyRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if r.OwnerType != OwnerTypeUser && r.OwnerType != OwnerTypeOrganization {
		return fmt.Errorf("owner_type must be either 'user' or 'organization'")
	}
	if r.ReferenceID == "" {
		return fmt.Errorf("reference_id is required")
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
	OwnerType   *string
	ReferenceID *string
	Page        int
	Limit       int
}

type GetApiKeyResponse struct {
	ApiKey *ApiKey `json:"api_key"`
}

type UpdateApiKeyRequest struct {
	Name                 *string         `json:"name,omitempty"`
	Enabled              *bool           `json:"enabled,omitempty"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
	RateLimitEnabled     *bool           `json:"rate_limit_enabled,omitempty"`
	RateLimitTimeWindow  *int            `json:"rate_limit_time_window,omitempty"`
	RateLimitMaxRequests *int            `json:"rate_limit_max_requests,omitempty"`
	RequestsRemaining    *int            `json:"requests_remaining,omitempty"`
	Permissions          []string        `json:"permissions,omitempty"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
}

func (r *UpdateApiKeyRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
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
		return fmt.Errorf("key is required")
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
