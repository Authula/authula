package handlers

import (
	"net/http"
	"strings"
)

func mapAdminErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))

	switch {
	case strings.Contains(message, "unauthorized"):
		return http.StatusUnauthorized
	case strings.Contains(message, "forbidden"):
		return http.StatusForbidden
	case strings.Contains(message, "not found"):
		return http.StatusNotFound
	case isAdminBadRequestMessage(message):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func isAdminBadRequestMessage(message string) bool {
	markers := []string{"required", "invalid", "cannot", "exceeds", "you can only", "no active", "already exists"}
	for _, marker := range markers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
