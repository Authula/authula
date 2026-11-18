package tests

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Authula/authula/openapi"
)

func TestMockOpenAPIService_AddOperation(t *testing.T) {
	t.Parallel()

	specErr := errors.New("add operation failed")

	tests := []struct {
		name    string
		method  string
		path    string
		opts    []openapi.OperationOption
		setup   func(m *MockOpenAPIService)
		wantErr error
	}{
		{
			name:   "returns nil when the operation is added",
			method: http.MethodGet,
			path:   "/api/v1/health",
			setup: func(m *MockOpenAPIService) {
				m.On("AddOperation", http.MethodGet, "/api/v1/health").Return(nil).Once()
			},
		},
		{
			name:   "returns the configured error",
			method: http.MethodPost,
			path:   "/api/v1/users",
			setup: func(m *MockOpenAPIService) {
				m.On("AddOperation", http.MethodPost, "/api/v1/users").Return(specErr).Once()
			},
			wantErr: specErr,
		},
		{
			name:   "matches expectations without operation options",
			method: http.MethodPut,
			path:   "/api/v1/users/{id}",
			opts: []openapi.OperationOption{
				openapi.WithSummary("Update a user"),
				openapi.WithTags("users"),
			},
			setup: func(m *MockOpenAPIService) {
				m.On("AddOperation", http.MethodPut, "/api/v1/users/{id}").Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := new(MockOpenAPIService)
			tt.setup(svc)

			err := svc.AddOperation(tt.method, tt.path, tt.opts...)

			assert.ErrorIs(t, err, tt.wantErr)
			svc.AssertExpectations(t)
		})
	}
}

func TestMockOpenAPIService_SpecJSON(t *testing.T) {
	t.Parallel()

	marshalErr := errors.New("marshal failed")

	tests := []struct {
		name    string
		spec    []byte
		err     error
		wantErr error
	}{
		{
			name: "returns the configured JSON spec",
			spec: []byte(`{"openapi":"3.0.3"}`),
		},
		{
			name:    "returns the configured marshal error",
			spec:    []byte(nil),
			err:     marshalErr,
			wantErr: marshalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := new(MockOpenAPIService)
			svc.On("SpecJSON").Return(tt.spec, tt.err).Once()

			got, err := svc.SpecJSON()

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.spec, got)
			svc.AssertExpectations(t)
		})
	}
}

func TestMockOpenAPIService_SpecYAML(t *testing.T) {
	t.Parallel()

	marshalErr := errors.New("marshal failed")

	tests := []struct {
		name    string
		spec    []byte
		err     error
		wantErr error
	}{
		{
			name: "returns the configured YAML spec",
			spec: []byte("openapi: 3.0.3\n"),
		},
		{
			name:    "returns the configured marshal error",
			spec:    []byte(nil),
			err:     marshalErr,
			wantErr: marshalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := new(MockOpenAPIService)
			svc.On("SpecYAML").Return(tt.spec, tt.err).Once()

			got, err := svc.SpecYAML()

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.spec, got)
			svc.AssertExpectations(t)
		})
	}
}

func TestMockOpenAPIService_RegisterSchema(t *testing.T) {
	t.Parallel()

	type userSchema struct {
		ID string `json:"id"`
	}

	schemaErr := errors.New("register schema failed")

	tests := []struct {
		name    string
		schema  string
		typ     any
		setup   func(m *MockOpenAPIService, name string, typ any)
		wantErr error
	}{
		{
			name:   "returns nil when the schema is registered",
			schema: "User",
			typ:    userSchema{},
			setup: func(m *MockOpenAPIService, name string, typ any) {
				m.On("RegisterSchema", name, typ).Return(nil).Once()
			},
		},
		{
			name:   "returns the configured error",
			schema: "User",
			typ:    userSchema{},
			setup: func(m *MockOpenAPIService, name string, typ any) {
				m.On("RegisterSchema", name, typ).Return(schemaErr).Once()
			},
			wantErr: schemaErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := new(MockOpenAPIService)
			tt.setup(svc, tt.schema, tt.typ)

			err := svc.RegisterSchema(tt.schema, tt.typ)

			assert.ErrorIs(t, err, tt.wantErr)
			svc.AssertExpectations(t)
		})
	}
}
