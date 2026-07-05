package emailtemplate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		def     Definition
		wantErr bool
	}{
		{
			name: "valid definition",
			def: Definition{
				Name:    "valid",
				Subject: "Hello {{.Name}}",
				Text:    "Hello {{.Name}}",
				HTML:    "<p>Hello {{.Name}}</p>",
			},
		},
		{
			name: "duplicate registration",
			def: Definition{
				Name:    "dup",
				Subject: "Subject",
				Text:    "Text",
				HTML:    "<p>HTML</p>",
			},
		},
		{
			name: "invalid subject template",
			def: Definition{
				Name:    "bad_subject",
				Subject: "{{.Name",
				Text:    "OK",
				HTML:    "<p>OK</p>",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			err := mgr.Register(tt.def)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Register a second time to test duplicate
				if tt.name == "duplicate registration" {
					err = mgr.Register(tt.def)
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestManager_Render(t *testing.T) {
	t.Parallel()

	type renderCase struct {
		name         string
		setup        func(*Manager)
		templateName string
		data         any
		wantSubject  string
		wantText     string
		wantHTML     string
		wantErr      bool
	}

	baseSetup := func(mgr *Manager) {
		_ = mgr.Register(Definition{
			Name:    "greeting",
			Subject: "Hello {{.Name}}",
			Text:    "Hello {{.Name}}, your code is {{.Code}}.",
			HTML:    "<p>Hello {{.Name}}, your code is <strong>{{.Code}}</strong>.</p>",
		})
	}

	tests := []renderCase{
		{
			name:         "renders template successfully",
			setup:        baseSetup,
			templateName: "greeting",
			data:         map[string]any{"Name": "Alice", "Code": "1234"},
			wantSubject:  "Hello Alice",
			wantText:     "Hello Alice, your code is 1234.",
			wantHTML:     "<p>Hello Alice, your code is <strong>1234</strong>.</p>",
		},
		{
			name:         "template not found",
			setup:        baseSetup,
			templateName: "nonexistent",
			data:         nil,
			wantErr:      true,
		},
		{
			name:         "HTML escapes dangerous content",
			setup:        baseSetup,
			templateName: "greeting",
			data:         map[string]any{"Name": "<script>alert('xss')</script>", "Code": "0000"},
			wantSubject:  "Hello <script>alert('xss')</script>",
			wantText:     "Hello <script>alert('xss')</script>, your code is 0000.",
			wantHTML:     "<p>Hello &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;, your code is <strong>0000</strong>.</p>",
		},
		{
			name: "missing key in HTML template errors",
			setup: func(mgr *Manager) {
				_ = mgr.Register(Definition{
					Name:    "missing_key",
					Subject: "{{.Name}}",
					Text:    "{{.Name}}",
					HTML:    "<p>{{.Name}}</p>",
				})
			},
			templateName: "missing_key",
			data:         map[string]any{"WrongKey": "value"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			tt.setup(mgr)

			subject, text, html, err := mgr.Render(tt.templateName, tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantSubject, subject)
			assert.Equal(t, tt.wantText, text)
			assert.Equal(t, tt.wantHTML, html)
		})
	}
}

func TestManager_Funcs(t *testing.T) {
	t.Parallel()

	type funcCase struct {
		name        string
		def         Definition
		data        any
		wantSubject string
		wantText    string
		wantHTML    string
	}

	tests := []funcCase{
		{
			name: "humanDuration with 1 hour",
			def: Definition{
				Name:    "human_dur",
				Subject: "Expires in {{humanDuration .Expiry}}",
				Text:    "Expires in {{humanDuration .Expiry}}",
				HTML:    "<p>Expires in {{humanDuration .Expiry}}</p>",
			},
			data:        map[string]any{"Expiry": time.Hour},
			wantSubject: "Expires in 1 hour",
			wantText:    "Expires in 1 hour",
			wantHTML:    "<p>Expires in 1 hour</p>",
		},
		{
			name: "humanDuration with 24 hours",
			def: Definition{
				Name:    "human_dur_days",
				Subject: "Expires in {{humanDuration .Expiry}}",
				Text:    "Expires in {{humanDuration .Expiry}}",
				HTML:    "<p>Expires in {{humanDuration .Expiry}}</p>",
			},
			data:        map[string]any{"Expiry": 24 * time.Hour},
			wantSubject: "Expires in 1 day",
			wantText:    "Expires in 1 day",
			wantHTML:    "<p>Expires in 1 day</p>",
		},
		{
			name: "plural singular",
			def: Definition{
				Name:    "plural_singular",
				Subject: `{{plural "item" .Count}}`,
				Text:    `{{plural "item" .Count}}`,
				HTML:    "<p>{{plural \"item\" .Count}}</p>",
			},
			data:        map[string]any{"Count": 1},
			wantSubject: "item",
			wantText:    "item",
			wantHTML:    "<p>item</p>",
		},
		{
			name: "plural multiple",
			def: Definition{
				Name:    "plural_multiple",
				Subject: `{{plural "item" .Count}}`,
				Text:    `{{plural "item" .Count}}`,
				HTML:    "<p>{{plural \"item\" .Count}}</p>",
			},
			data:        map[string]any{"Count": 3},
			wantSubject: "items",
			wantText:    "items",
			wantHTML:    "<p>items</p>",
		},
		{
			name: "upper and lower",
			def: Definition{
				Name:    "case",
				Subject: "{{upper .V}} {{lower .V}}",
				Text:    "{{upper .V}} {{lower .V}}",
				HTML:    "<p>{{upper .V}} {{lower .V}}</p>",
			},
			data:        map[string]any{"V": "Hello"},
			wantSubject: "HELLO hello",
			wantText:    "HELLO hello",
			wantHTML:    "<p>HELLO hello</p>",
		},
		{
			name: "currentYear",
			def: Definition{
				Name:    "year",
				Subject: "{{currentYear}}",
				Text:    "{{currentYear}}",
				HTML:    "<p>{{currentYear}}</p>",
			},
			data:        nil,
			wantSubject: "2026",
			wantText:    "2026",
			wantHTML:    "<p>2026</p>",
		},
		{
			name: "title function",
			def: Definition{
				Name:    "title_test",
				Subject: "{{title .V}}",
				Text:    "{{title .V}}",
				HTML:    "<p>{{title .V}}</p>",
			},
			data:        map[string]any{"V": "hello"},
			wantSubject: "Hello",
			wantText:    "Hello",
			wantHTML:    "<p>Hello</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mgr := NewManager()
			err := mgr.Register(tt.def)
			assert.NoError(t, err)

			subject, text, html, err := mgr.Render(tt.def.Name, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSubject, subject)
			assert.Equal(t, tt.wantText, text)
			assert.Equal(t, tt.wantHTML, html)
		})
	}
}
