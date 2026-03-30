package repositories

import (
	"fmt"
	"strings"

	"github.com/Authula/authula/plugins/access-control/constants"
)

func wrapRepositoryError(action string, err error) error {
	if err == nil {
		return nil
	}

	if isUniqueConstraintError(err) {
		return constants.ErrConflict
	}

	return fmt.Errorf("failed to %s: %w", action, err)
}

func isUniqueConstraintError(err error) bool {
	message := strings.ToLower(err.Error())

	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "unique violation") ||
		strings.Contains(message, "duplicate key value") ||
		strings.Contains(message, "duplicate entry") ||
		strings.Contains(message, "error 1062")
}
