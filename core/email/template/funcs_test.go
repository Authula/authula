package emailtemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string returns empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single character is uppercased",
			input: "a",
			want:  "A",
		},
		{
			name:  "lowercase first letter is uppercased",
			input: "hello",
			want:  "Hello",
		},
		{
			name:  "rest of the string is unchanged",
			input: "hello WORLD",
			want:  "Hello WORLD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Title(tt.input)

			assert.Equal(t, tt.want, got)
		})
	}
}
