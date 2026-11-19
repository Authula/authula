package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		existing  string
		requested string
		want      string
	}{
		{
			name:      "adds new scopes",
			existing:  "openid profile",
			requested: "email",
			want:      "openid profile email",
		},
		{
			name:      "removes duplicates",
			existing:  "openid profile",
			requested: "profile email openid",
			want:      "openid profile email",
		},
		{
			name:      "handles empty inputs",
			existing:  "",
			requested: "",
			want:      "",
		},
		{
			name:      "handles extra whitespace",
			existing:  " openid  profile ",
			requested: " email ",
			want:      "openid profile email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MergeScopes(tt.existing, tt.requested))
		})
	}
}
