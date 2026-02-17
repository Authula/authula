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
