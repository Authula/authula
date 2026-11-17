package openapi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/openapi"
)

type recordedOperation struct {
	method string
	path   string
	opts   int
}

type fakeOpenAPIService struct {
	operations []recordedOperation
	failOn     string
}

func (f *fakeOpenAPIService) AddOperation(method, pathPattern string, opts ...openapi.OperationOption) error {
	f.operations = append(f.operations, recordedOperation{method: method, path: pathPattern, opts: len(opts)})
	if f.failOn == method+" "+pathPattern {
		return errors.New("boom")
	}
	return nil
}

func (f *fakeOpenAPIService) SpecJSON() ([]byte, error) { return nil, nil }
func (f *fakeOpenAPIService) SpecYAML() ([]byte, error) { return nil, nil }
func (f *fakeOpenAPIService) RegisterSchema(string, any) error {
	return nil
}

func TestRegisterOpenAPIDocs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		failOn                string
		wantErr               bool
		wantOperationCount    int
		wantFirstMethod       string
		wantFirstPath         string
		wantMinOptionsPerCall int
	}{
		{
			name:                  "registers all operations",
			wantOperationCount:    22,
			wantFirstMethod:       http.MethodPost,
			wantFirstPath:         "/access-control/roles",
			wantMinOptionsPerCall: 5,
		},
		{
			name:                  "propagates registration error",
			failOn:                http.MethodPost + " /access-control/roles",
			wantErr:               true,
			wantOperationCount:    22,
			wantFirstMethod:       http.MethodPost,
			wantFirstPath:         "/access-control/roles",
			wantMinOptionsPerCall: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeOpenAPIService{failOn: tc.failOn}

			err := RegisterOpenAPIDocs(service)

			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "boom")
			} else {
				require.NoError(t, err)
			}
			require.Len(t, service.operations, tc.wantOperationCount)
			assert.Equal(t, tc.wantFirstMethod, service.operations[0].method)
			assert.Equal(t, tc.wantFirstPath, service.operations[0].path)
			for _, operation := range service.operations {
				assert.GreaterOrEqual(t, operation.opts, tc.wantMinOptionsPerCall)
			}
		})
	}
}
