package email_password

import (
	"testing"

	"github.com/stretchr/testify/require"

	emailconstants "github.com/Authula/authula/core/email/constants"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestNewEmailPasswordEmailTemplateManager(t *testing.T) {
	t.Parallel()

	manager, err := newEmailPasswordEmailTemplateManager()
	require.NoError(t, err)

	_, text, _, err := manager.Render(emailconstants.VerifyEmailEmailTemplateName, types.VerifyEmailContext{
		VerificationLink: "https://example.com/verify?token=abc",
	})
	require.NoError(t, err)
	require.Contains(t, text, "https://example.com/verify?token=abc")
}

func TestNewEmailPasswordEmailTemplateManager_InvalidEnvOverride(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantErr string
	}{
		{
			name:    "fails when the subject override does not parse",
			envVar:  constants.EnvVerifyEmailSubject,
			wantErr: `failed to parse subject template "verify_email"`,
		},
		{
			name:    "fails when the text override does not parse",
			envVar:  constants.EnvVerifyEmailText,
			wantErr: `failed to parse text template "verify_email"`,
		},
		{
			name:    "fails when the html override does not parse",
			envVar:  constants.EnvVerifyEmailHTML,
			wantErr: `failed to parse html template "verify_email"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, "{{")

			manager, err := newEmailPasswordEmailTemplateManager()

			require.ErrorContains(t, err, tt.wantErr)
			require.Nil(t, manager)
		})
	}
}

func TestMustLoadTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		wantContains string
	}{
		{
			name:         "loads an embedded subject template",
			path:         "templates/verify_email/subject.tmpl",
			wantContains: "Verify your email",
		},
		{
			name:         "loads an embedded text template",
			path:         "templates/verify_email/text.tmpl",
			wantContains: "{{.VerificationLink}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Contains(t, mustLoadTemplate(tt.path), tt.wantContains)
		})
	}
}

func TestMustLoadTemplate_PanicsWhenFileIsMissing(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		mustLoadTemplate("templates/missing/subject.tmpl")
	})
}
