package models

const (
	ContextAccessControlAssignRole ContextKey = "access_control.assign_role"
)

// Access Control

type AccessControlAssignRoleContext struct {
	UserID         string
	RoleName       string
	AssignerUserID *string
}
