package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString_NegativeLength(t *testing.T) {
	t.Parallel()

	got, err := GenerateRandomString(-1)

	require.Error(t, err)
	assert.Empty(t, got)
}
