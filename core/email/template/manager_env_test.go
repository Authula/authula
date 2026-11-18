package emailtemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This test is not parallel because t.Setenv cannot be used with t.Parallel.
// Each case uses its own template name so environment overrides do not leak between cases.
func TestManager_RegisterEnvOverrides(t *testing.T) {
	tests := []struct {
		name        string
		def         Definition
		env         map[string]string
		wantErr     bool
		wantSubject string
		wantText    string
		wantHTML    string
	}{
		{
			name: "subject override replaces definition subject",
			def: Definition{
				Name:    "env_subject",
				Subject: "Default {{.Name}}",
				Text:    "Hello {{.Name}}",
				HTML:    "<p>Hello {{.Name}}</p>",
			},
			env:         map[string]string{"AUTHULA_TEMPLATE_ENV_SUBJECT_SUBJECT": "Override {{.Name}}"},
			wantSubject: "Override Alice",
			wantText:    "Hello Alice",
			wantHTML:    "<p>Hello Alice</p>",
		},
		{
			name: "text override replaces definition text",
			def: Definition{
				Name:    "env_text",
				Subject: "Hello {{.Name}}",
				Text:    "Default {{.Name}}",
				HTML:    "<p>Hello {{.Name}}</p>",
			},
			env:         map[string]string{"AUTHULA_TEMPLATE_ENV_TEXT_TEXT": "Override {{.Name}}"},
			wantSubject: "Hello Alice",
			wantText:    "Override Alice",
			wantHTML:    "<p>Hello Alice</p>",
		},
		{
			name: "html override replaces definition html",
			def: Definition{
				Name:    "env_html",
				Subject: "Hello {{.Name}}",
				Text:    "Hello {{.Name}}",
				HTML:    "<p>Default {{.Name}}</p>",
			},
			env:         map[string]string{"AUTHULA_TEMPLATE_ENV_HTML_HTML": "<p>Override {{.Name}}</p>"},
			wantSubject: "Hello Alice",
			wantText:    "Hello Alice",
			wantHTML:    "<p>Override Alice</p>",
		},
		{
			name: "empty override keeps definition subject",
			def: Definition{
				Name:    "env_empty",
				Subject: "Default {{.Name}}",
				Text:    "Hello {{.Name}}",
				HTML:    "<p>Hello {{.Name}}</p>",
			},
			env:         map[string]string{"AUTHULA_TEMPLATE_ENV_EMPTY_SUBJECT": ""},
			wantSubject: "Default Alice",
			wantText:    "Hello Alice",
			wantHTML:    "<p>Hello Alice</p>",
		},
		{
			name: "invalid subject override returns parse error",
			def: Definition{
				Name:    "env_bad_subject",
				Subject: "Default",
				Text:    "Hello",
				HTML:    "<p>Hello</p>",
			},
			env:     map[string]string{"AUTHULA_TEMPLATE_ENV_BAD_SUBJECT_SUBJECT": "{{.Name"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			mgr := NewManager()
			err := mgr.Register(tt.def)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			subject, text, html, err := mgr.Render(tt.def.Name, map[string]any{"Name": "Alice"})
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSubject, subject)
			assert.Equal(t, tt.wantText, text)
			assert.Equal(t, tt.wantHTML, html)
		})
	}
}
