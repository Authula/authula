package openapi_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
	adminopenapi "github.com/Authula/authula/plugins/admin/openapi"
)

type recordingOpenAPIService struct {
	operations []string
	failOn     string
	err        error
}

func (s *recordingOpenAPIService) AddOperation(method, pathPattern string, _ ...openapi.OperationOption) error {
	key := method + " " + pathPattern
	s.operations = append(s.operations, key)
	if key == s.failOn {
		return s.err
	}
	return nil
}

func (s *recordingOpenAPIService) SpecJSON() ([]byte, error) { return nil, nil }

func (s *recordingOpenAPIService) SpecYAML() ([]byte, error) { return nil, nil }

func (s *recordingOpenAPIService) RegisterSchema(string, any) error { return nil }

func TestRegisterOpenAPIDocs(t *testing.T) {
	t.Parallel()

	svc := &recordingOpenAPIService{}
	require.NoError(t, adminopenapi.RegisterOpenAPIDocs(svc))

	require.Len(t, svc.operations, 28)
	assert.Equal(t, http.MethodPost+" /admin/users", svc.operations[0])
	assert.Contains(t, svc.operations, http.MethodPost+" /admin/impersonations/{impersonation_id}/stop")
	assert.Contains(t, svc.operations, http.MethodGet+" /admin/sessions/states/revoked")
}

func TestRegisterOpenAPIDocs_ReturnsJoinedError(t *testing.T) {
	t.Parallel()

	addErr := errors.New("add operation failed")
	svc := &recordingOpenAPIService{
		failOn: http.MethodGet + " /admin/users",
		err:    addErr,
	}

	err := adminopenapi.RegisterOpenAPIDocs(svc)
	require.Error(t, err)
	assert.ErrorIs(t, err, addErr)
	// Errors are joined, so registration continues past the failing operation.
	assert.Contains(t, svc.operations, http.MethodPost+" /admin/impersonations")
}
