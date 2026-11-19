package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSigningKeyAlgorithm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		algorithm string
		wantErr   error
	}{
		{name: "RS256 is supported", algorithm: SigningKeyAlgorithmRS256},
		{name: "ES256 is supported", algorithm: SigningKeyAlgorithmES256},
		{name: "PS256 is supported", algorithm: SigningKeyAlgorithmPS256},
		{name: "EdDSA is rejected", algorithm: "EdDSA", wantErr: ErrUnsupportedSigningAlgorithm},
		{name: "HS256 is rejected", algorithm: "HS256", wantErr: ErrUnsupportedSigningAlgorithm},
		{name: "empty is rejected", algorithm: "", wantErr: ErrUnsupportedSigningAlgorithm},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSigningKeyAlgorithm(tt.algorithm)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateSigningKeyPurpose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		purpose string
		wantErr error
	}{
		{name: "jwt plugin purpose", purpose: SigningKeyPurposeJWTPlugin},
		{name: "oidc id token purpose", purpose: SigningKeyPurposeOIDCIDToken},
		{name: "oidc access token purpose", purpose: SigningKeyPurposeOIDCAccessToken},
		{name: "unknown purpose is rejected", purpose: "other", wantErr: ErrInvalidSigningKeyPurpose},
		{name: "empty purpose is rejected", purpose: "", wantErr: ErrInvalidSigningKeyPurpose},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSigningKeyPurpose(tt.purpose)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSigningKeyDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SigningKeyAlgorithmRS256, DefaultSigningKeyAlgorithm)
	assert.Equal(t, 2048, DefaultRSAKeySize)
}
