package util

import (
	"encoding/json"
	"net/http"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type requestContextProvider interface {
	GetRequestContext() *models.RequestContext
}

func ParseJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(dest)
}

func JSONResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
