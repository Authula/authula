package tests

import (
	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/openapi"
)

// MockOpenAPIService is a testify mock of openapi.OpenAPIService.
type MockOpenAPIService struct {
	mock.Mock
}

var _ openapi.OpenAPIService = (*MockOpenAPIService)(nil)

func (m *MockOpenAPIService) AddOperation(method, pathPattern string, _ ...openapi.OperationOption) error {
	args := m.Called(method, pathPattern)
	return args.Error(0)
}

func (m *MockOpenAPIService) SpecJSON() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockOpenAPIService) SpecYAML() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockOpenAPIService) RegisterSchema(name string, typ any) error {
	args := m.Called(name, typ)
	return args.Error(0)
}
