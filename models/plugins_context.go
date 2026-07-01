package models

// Access Control

type AccessControlAssignRoleContext struct {
	UserID         string
	RoleName       string
	AssignerUserID *string
}

// Rate Limit

type RateLimitRuleContext struct {
	Key           string
	WindowSeconds int
	MaxRequests   int
}
