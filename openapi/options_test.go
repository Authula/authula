package openapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	swaggestopenapi "github.com/swaggest/openapi-go"

	"github.com/Authula/authula/openapi"
)

func TestWithJSONContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "sets application/json on an empty content unit",
			contentType: "",
		},
		{
			name:        "replaces a non-JSON content type with application/json",
			contentType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			unit := new(swaggestopenapi.ContentUnit)
			unit.ContentType = tt.contentType

			// Act
			openapi.WithJSONContentType()(unit)

			// Assert
			assert.Equal(t, "application/json", unit.ContentType)
		})
	}
}
