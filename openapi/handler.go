package openapi

import (
	"net/http"
)

func NewOpenAPISpecHandler(svc OpenAPIService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec, err := svc.SpecJSON()
		if err != nil {
			http.Error(w, "Failed to marshal OpenAPI spec: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(spec)
	}
}

func NewOpenAPISpecYAMLHandler(svc OpenAPIService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec, err := svc.SpecYAML()
		if err != nil {
			http.Error(w, "Failed to marshal OpenAPI spec: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/vnd.yaml")
		_, _ = w.Write(spec)
	}
}
