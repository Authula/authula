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
		{
			name:  "multi-byte leading character",
			input: "élan",
			want:  "Élan",
		},
		{
			name:  "multi-word ASCII only capitalises first character",
			input: "hello world",
			want:  "Hello world",
		},
		{
			name:  "multi-byte leading character in each word",
			input: "élan vital über café",
			want:  "Élan vital über café",
		},
		{
			name:  "multi-byte leading character in later word",
			input: "hello élan",
			want:  "Hello élan",
		},
		{
			name:  "Cyrillic leading character",
			input: "привет мир",
			want:  "Привет мир",
		},
		{
			name:  "non-cased script is unchanged",
			input: "日本語",
			want:  "日本語",
		},
		{
			name:  "invalid leading byte is left untouched",
			input: "\xfflan",
			want:  "\xfflan",
		},
		{
			name:  "literal replacement character is preserved",
			input: "\uFFFDlan",
			want:  "\uFFFDlan",
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
