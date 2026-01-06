package events

// AuditMetadata contains optional audit information for security events
type AuditMetadata struct {
	ClientIP  string `json:"client_ip,omitempty"`  // IP address of the client (may be masked)
	UserAgent string `json:"user_agent,omitempty"` // User agent string
	DeviceID  string `json:"device_id,omitempty"`  // Device identifier for session tracking
}

// TokenReuseRecoveredEvent emitted when a token is reused within the grace period (first occurrence)
type TokenReuseRecoveredEvent struct {
	Type              string        `json:"type"`
	SessionID         string        `json:"session_id"`
	TokenHash         string        `json:"token_hash"`
	DeltaMs           int64         `json:"delta_ms"`            // Milliseconds between revocation and reuse
	GracePeriodConfig string        `json:"grace_period_config"` // Configured grace period as string
	Metadata          AuditMetadata `json:"metadata"`
	Timestamp         string        `json:"timestamp"`
}

// TokenReuseThrottledEvent emitted when a token is reused multiple times within the grace period
type TokenReuseThrottledEvent struct {
	Type              string        `json:"type"`
	SessionID         string        `json:"session_id"`
	TokenHash         string        `json:"token_hash"`
	DeltaMs           int64         `json:"delta_ms"`            // Milliseconds between revocation and reuse
	GracePeriodConfig string        `json:"grace_period_config"` // Configured grace period as string
	AttemptCount      int           `json:"attempt_count"`       // Number of reuse attempts
	Metadata          AuditMetadata `json:"metadata"`
	Timestamp         string        `json:"timestamp"`
}

// TokenReuseMaliciousEvent emitted when a token is reused after the grace period expires (potential attack)
type TokenReuseMaliciousEvent struct {
	Type              string        `json:"type"`
	SessionID         string        `json:"session_id"`
	TokenHash         string        `json:"token_hash"`
	DeltaMs           int64         `json:"delta_ms"`            // Milliseconds between revocation and reuse
	GracePeriodConfig string        `json:"grace_period_config"` // Configured grace period as string
	Metadata          AuditMetadata `json:"metadata"`
	Timestamp         string        `json:"timestamp"`
}
