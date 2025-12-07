package domain

type ProviderType string

const (
	ProviderEmail   ProviderType = "email"
	ProviderDiscord ProviderType = "discord"
	ProviderGitHub  ProviderType = "github"
	ProviderGoogle  ProviderType = "google"
)
