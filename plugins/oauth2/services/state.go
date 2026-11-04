package services

import (
	"encoding/json"
	"fmt"
	"time"
)

// StatePurpose identifies why an OAuth2 flow was started.
type StatePurpose string

const (
	StatePurposeLogin StatePurpose = "login"
	StatePurposeLink  StatePurpose = "link"
)

// StatePayload is the signed, provider-bound data stored in the OAuth2 state cookie.
type StatePayload struct {
	State    string       `json:"state"`
	Provider string       `json:"provider"`
	UserID   string       `json:"user_id,omitempty"`
	Purpose  StatePurpose `json:"purpose"`
}

// SignState serialises and signs a state payload.
func SignState(payload StatePayload, secret []byte) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode oauth state: %w", err)
	}
	return SignCookie(string(data), secret)
}

// ParseState validates the signature and age of a signed state cookie.
func ParseState(signed string, secret []byte, maxAge time.Duration) (*StatePayload, error) {
	raw, err := ValidateCookie(signed, secret, maxAge)
	if err != nil {
		return nil, err
	}

	var payload StatePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid oauth state payload: %w", err)
	}
	if payload.State == "" || payload.Provider == "" {
		return nil, fmt.Errorf("incomplete oauth state payload")
	}
	if payload.Purpose != StatePurposeLogin && payload.Purpose != StatePurposeLink {
		return nil, fmt.Errorf("invalid oauth state purpose")
	}

	return &payload, nil
}
