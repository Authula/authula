package constants

import "errors"

var (
	ErrCannotUpdateSystemRole                  = errors.New("cannot update system role")
	ErrAtleastOneFieldRequiredToUpdateResource = errors.New("at least one field is required to update the resource")
)
