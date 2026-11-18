package emailtemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCommonContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appName string
		baseURL string
		want    CommonContext
	}{
		{
			name:    "sets app name and base URL",
			appName: "Authula",
			baseURL: "https://auth.example.com",
			want: CommonContext{
				AppName: "Authula",
				BaseURL: "https://auth.example.com",
			},
		},
		{
			name: "keeps empty values",
			want: CommonContext{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewCommonContext(tt.appName, tt.baseURL)

			assert.Equal(t, tt.want, got)
		})
	}
}
