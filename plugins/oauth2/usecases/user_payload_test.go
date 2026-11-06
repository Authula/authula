package usecases

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestCallbackUseCase_applyUserPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userInfo types.UserInfo
		payload  string
		want     types.UserInfo
		wantLog  string
	}{
		{
			name:     "empty payload leaves user info untouched",
			userInfo: types.UserInfo{Name: "Existing Name", Email: "existing@example.com"},
			payload:  "",
			want:     types.UserInfo{Name: "Existing Name", Email: "existing@example.com"},
		},
		{
			name:     "payload name fills an empty name",
			userInfo: types.UserInfo{Email: "existing@example.com"},
			payload:  `{"name":{"firstName":"Ada","lastName":"Lovelace"}}`,
			want:     types.UserInfo{Name: "Ada Lovelace", Email: "existing@example.com"},
		},
		{
			name:     "existing name is not overwritten by the payload",
			userInfo: types.UserInfo{Name: "Provider Name"},
			payload:  `{"name":{"firstName":"Ada","lastName":"Lovelace"}}`,
			want:     types.UserInfo{Name: "Provider Name"},
		},
		{
			name:     "payload email is ignored",
			userInfo: types.UserInfo{},
			payload:  `{"email":"attacker@example.com"}`,
			want:     types.UserInfo{},
		},
		{
			name:     "payload email does not replace the provider email",
			userInfo: types.UserInfo{Email: "provider@example.com"},
			payload:  `{"name":{"firstName":"Ada"},"email":"attacker@example.com"}`,
			want:     types.UserInfo{Name: "Ada", Email: "provider@example.com"},
		},
		{
			name:     "payload without a name leaves the name empty",
			userInfo: types.UserInfo{Email: "existing@example.com"},
			payload:  `{"email":"ada@example.com"}`,
			want:     types.UserInfo{Email: "existing@example.com"},
		},
		{
			name:     "malformed payload is logged and ignored",
			userInfo: types.UserInfo{Name: "Existing Name", Email: "existing@example.com"},
			payload:  `{"name":`,
			want:     types.UserInfo{Name: "Existing Name", Email: "existing@example.com"},
			wantLog:  "failed to parse provider user payload",
		},
		{
			name:     "payload name is sanitised",
			userInfo: types.UserInfo{},
			payload:  `{"name":{"firstName":"Ada\u0000Byron","lastName":"   Love   lace"}}`,
			want:     types.UserInfo{Name: "AdaByron Love lace"},
		},
		{
			name:     "payload name is truncated to 100 runes",
			userInfo: types.UserInfo{},
			payload:  `{"name":{"firstName":"` + strings.Repeat("a", 500) + `"}}`,
			want:     types.UserInfo{Name: strings.Repeat("a", 100)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newCallbackFixture(t, &oauth2tests.MockProvider{ProviderName: "google"})
			logger := &internaltests.LevelRecordingLogger{}
			f.useCase.Logger = logger

			userInfo := tt.userInfo
			f.useCase.applyUserPayload(&userInfo, tt.payload)

			is := assert.New(t)
			is.Equal(tt.want.Name, userInfo.Name)
			is.Equal(tt.want.Email, userInfo.Email)

			records := logger.Records()
			if tt.wantLog == "" {
				is.Empty(records)
				return
			}

			require.Len(t, records, 1)
			is.Equal("warn", records[0].Level)
			is.Contains(records[0].Message, tt.wantLog)
		})
	}
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "control characters are stripped",
			input: "Ada\x00Byron\x07",
			want:  "AdaByron",
		},
		{
			name:  "tabs become a single space",
			input: "Ada\tLovelace",
			want:  "Ada Lovelace",
		},
		{
			name:  "newlines become a single space",
			input: "Ada\nLovelace",
			want:  "Ada Lovelace",
		},
		{
			name:  "crlf becomes a single space",
			input: "Ada\r\nLovelace",
			want:  "Ada Lovelace",
		},
		{
			name:  "repeated whitespace is collapsed",
			input: "Ada \t Lovelace",
			want:  "Ada Lovelace",
		},
		{
			name:  "leading and trailing space is trimmed",
			input: "  Ada Lovelace  ",
			want:  "Ada Lovelace",
		},
		{
			name:  "empty string stays empty",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only becomes empty",
			input: " \t\n ",
			want:  "",
		},
		{
			name:  "over length name is truncated to 100 runes",
			input: strings.Repeat("a", 500),
			want:  strings.Repeat("a", 100),
		},
		{
			name:  "truncation does not split multi-byte runes",
			input: strings.Repeat("é", 150),
			want:  strings.Repeat("é", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := sanitizeName(tt.input)

			is := assert.New(t)
			is.Equal(tt.want, got)
			is.True(utf8.ValidString(got), "sanitised name must remain valid UTF-8")
			is.LessOrEqual(utf8.RuneCountInString(got), maxUserNameLength)
		})
	}
}
