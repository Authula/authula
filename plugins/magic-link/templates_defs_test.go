package magiclink

import (
	"testing"

	magiclinkconstants "github.com/Authula/authula/plugins/magic-link/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMagicLinkEmailTemplateManager(t *testing.T) {
	mgr, err := newMagicLinkEmailTemplateManager()

	require.NoError(t, err)
	require.NotNil(t, mgr)
}

func TestMustLoadMagicLinkTemplate(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantPart string
	}{
		{
			name:     "subject template",
			path:     "templates/magic_link_sign_in/subject.tmpl",
			wantPart: "magic link",
		},
		{
			name:     "text template",
			path:     "templates/magic_link_sign_in/text.tmpl",
			wantPart: "{{.MagicLink}}",
		},
		{
			name:     "html template",
			path:     "templates/magic_link_sign_in/html.tmpl",
			wantPart: "{{.MagicLink}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := mustLoadMagicLinkTemplate(tt.path)

			assert.NotEmpty(t, content)
			assert.Contains(t, content, tt.wantPart)
		})
	}
}

func TestMustLoadMagicLinkTemplate_PanicsOnMissingTemplate(t *testing.T) {
	assert.Panics(t, func() {
		mustLoadMagicLinkTemplate("templates/magic_link_sign_in/missing.tmpl")
	})
}

func TestNewMagicLinkEmailTemplateManager_InvalidTemplate(t *testing.T) {
	t.Setenv(magiclinkconstants.EnvMagicLinkSignInHTML, "{{.Broken")

	mgr, err := newMagicLinkEmailTemplateManager()

	require.ErrorContains(t, err, "failed to parse html template")
	assert.Nil(t, mgr)
}
