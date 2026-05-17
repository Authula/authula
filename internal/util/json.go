package util

import (
	"encoding/json"
	"net/http"
)

func ParseJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(dest)
}

func JSONResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Converts a Go struct or map into a map[string]any using JSON tags for field mapping.
func ToMap(v any) map[string]any {
	if v == nil {
		return nil
	}

	// Performance optimization: if it's already a map, just use it
	if m, ok := v.(map[string]any); ok {
		return m
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var res map[string]any
	_ = json.Unmarshal(data, &res)
	return res
}

// Converts a map[string]any back into a specific Go struct pointer.
// The 'dest' argument must be a pointer to a struct (e.g., &models.User{}).
func FromMap(m map[string]any, dest any) error {
	if m == nil {
		return nil
	}

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}
