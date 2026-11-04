package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

const (
	defaultGitHubUserInfoURL = "https://api.github.com/user"
	defaultGitHubEmailsURL   = "https://api.github.com/user/emails"
)

type GitHubOAuth2Provider struct {
	*BaseProvider
	userInfoURL string
	emailsURL   string
	fields      providerFields
}

var githubDefaults = ProviderDefaults{
	Endpoint:    github.Endpoint,
	Scopes:      []string{"read:user", "user:email"},
	UserInfoURL: defaultGitHubUserInfoURL,
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewGitHubOAuth2Provider(cfg types.ProviderConfig, opts *ProviderOptions) *GitHubOAuth2Provider {
	return &GitHubOAuth2Provider{
		BaseProvider: NewBaseProvider("github", ResolveOAuth2Config(cfg, githubDefaults), opts),
		userInfoURL:  ResolveUserInfoURL(cfg, defaultGitHubUserInfoURL),
		emailsURL:    defaultGitHubEmailsURL,
		fields: providerFields{
			userID:  defaultField(cfg.UserIDField, "id"),
			email:   defaultField(cfg.EmailField, "email"),
			name:    defaultField(cfg.NameField, "name"),
			picture: defaultField(cfg.PictureField, "avatar_url"),
		},
	}
}

func (p *GitHubOAuth2Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}

	id, err := GetIDField(info, p.fields.userID)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub account id: %w", err)
	}

	profileEmail := util.NormalizeEmail(GetStringField(info, p.fields.email))
	email, verified, err := p.resolveEmail(ctx, token, profileEmail)
	if err != nil {
		return nil, err
	}

	name := GetStringField(info, p.fields.name)
	if name == "" {
		name = GetStringField(info, "login")
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             email,
		EmailVerified:     verified,
		Name:              name,
		Picture:           GetStringField(info, p.fields.picture),
		Raw:               info,
	}, nil
}

// resolveEmail prefers a verified address from /user/emails. The profile email is used
// as a fallback and is marked unverified.
func (p *GitHubOAuth2Provider) resolveEmail(ctx context.Context, token *oauth2.Token, profileEmail string) (string, bool, error) {
	emails, err := p.listEmails(ctx, token)
	if err != nil {
		if profileEmail != "" {
			return profileEmail, false, nil
		}
		return "", false, err
	}

	verifiedEmail, verified := selectVerifiedEmail(emails)
	if verified {
		return verifiedEmail, true, nil
	}

	if profileEmail != "" {
		return profileEmail, false, nil
	}
	if len(emails) > 0 {
		return util.NormalizeEmail(emails[0].Email), false, nil
	}

	return "", false, nil
}

func selectVerifiedEmail(emails []githubEmail) (string, bool) {
	for _, email := range emails {
		if email.Verified && email.Primary {
			return util.NormalizeEmail(email.Email), true
		}
	}
	for _, email := range emails {
		if email.Verified {
			return util.NormalizeEmail(email.Email), true
		}
	}
	return "", false
}

func (p *GitHubOAuth2Provider) listEmails(ctx context.Context, token *oauth2.Token) ([]githubEmail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.emailsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub email request: %w", err)
	}

	token.SetAuthHeader(req)

	resp, err := p.opts.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub emails: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("failed to fetch GitHub emails: empty response")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && p.opts.logger() != nil {
			p.opts.logger().Debug("failed to close GitHub emails response", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch GitHub emails: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUserInfoResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub emails: %w", err)
	}
	if len(body) > maxUserInfoResponseBytes {
		return nil, fmt.Errorf("GitHub emails response exceeds %d bytes", maxUserInfoResponseBytes)
	}

	var emails []githubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub emails: %w", err)
	}

	return emails, nil
}

func (p *GitHubOAuth2Provider) RequiresPKCE() bool {
	return true
}
