package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coretypes "github.com/Authula/authula/core/types"
)

func TestSignOutRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sessionID  *string
		signOutAll *bool
		wantErr    string
	}{
		{
			name:       "no fields set",
			sessionID:  nil,
			signOutAll: nil,
		},
		{
			name:       "sign out all only",
			sessionID:  nil,
			signOutAll: new(true),
		},
		{
			name:      "session id only",
			sessionID: new("session-1"),
		},
		{
			name:       "session id with sign out all",
			sessionID:  new("session-1"),
			signOutAll: new(false),
		},
		{
			name:      "session id padded with spaces",
			sessionID: new(" session-1 "),
		},
		{
			name:      "empty session id",
			sessionID: new(""),
			wantErr:   "session_id cannot be empty if provided",
		},
		{
			name:      "whitespace only session id",
			sessionID: new(" \t\n"),
			wantErr:   "session_id cannot be empty if provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := &coretypes.SignOutRequest{
				SessionID:  tt.sessionID,
				SignOutAll: tt.signOutAll,
			}

			err := req.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestSignOutRequest_Validate_DoesNotMutateRequest(t *testing.T) {
	t.Parallel()

	req := &coretypes.SignOutRequest{
		SessionID:  new("  "),
		SignOutAll: new(true),
	}

	require.Error(t, req.Validate())
	assert.Equal(t, "  ", *req.SessionID)
	assert.True(t, *req.SignOutAll)
}
