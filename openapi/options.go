package openapi

import (
	swaggestopenapi "github.com/swaggest/openapi-go"
)

type OperationOption func(swaggestopenapi.OperationContext)

func WithSummary(summary string) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.SetSummary(summary)
	}
}

func WithDescription(description string) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.SetDescription(description)
	}
}

func WithTags(tags ...string) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.SetTags(tags...)
	}
}

func WithOperationID(id string) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.SetID(id)
	}
}

func WithDeprecated(deprecated bool) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.SetIsDeprecated(deprecated)
	}
}

func WithRequest(req any, opts ...swaggestopenapi.ContentOption) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.AddReqStructure(req, opts...)
	}
}

func WithResponse(resp any, opts ...swaggestopenapi.ContentOption) OperationOption {
	return func(oc swaggestopenapi.OperationContext) {
		oc.AddRespStructure(resp, opts...)
	}
}

// WithResponseStatus is a convenience wrapper that combines WithResponse
// and WithHTTPStatus into a single option.
func WithResponseStatus(status int, resp any) OperationOption {
	return WithResponse(resp, swaggestopenapi.WithHTTPStatus(status))
}

// WithJSONContentType sets the content type to application/json on a response.
func WithJSONContentType() swaggestopenapi.ContentOption {
	return swaggestopenapi.WithContentType("application/json")
}
