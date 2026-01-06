package models

type AuthProviderID string

const (
	AuthProviderEmail   AuthProviderID = "email"
	AuthProviderDiscord AuthProviderID = "discord"
	AuthProviderGitHub  AuthProviderID = "github"
	AuthProviderGoogle  AuthProviderID = "google"
)

func (id AuthProviderID) String() string {
	return string(id)
}
