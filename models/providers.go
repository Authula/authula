package models

type AuthProviderID string

const (
	AuthProviderEmail     AuthProviderID = "email"
	AuthProviderMagicLink AuthProviderID = "magic-link"
	AuthProviderDiscord   AuthProviderID = "discord"
	AuthProviderDropbox   AuthProviderID = "dropbox"
	AuthProviderGitHub    AuthProviderID = "github"
	AuthProviderGoogle    AuthProviderID = "google"
	AuthProviderMicrosoft AuthProviderID = "microsoft"
)

func (id AuthProviderID) String() string {
	return string(id)
}
