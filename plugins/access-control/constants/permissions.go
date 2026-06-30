package constants

// Access control permissions
const (
	RolesCreatePermission                   = "access-control:roles:create"
	RolesListPermission                     = "access-control:roles:list"
	RolesReadPermission                     = "access-control:roles:read"
	RolesUpdatePermission                   = "access-control:roles:update"
	RolesDeletePermission                   = "access-control:roles:delete"
	PermissionsCreatePermission             = "access-control:permissions:create"
	PermissionsListPermission               = "access-control:permissions:list"
	PermissionsReadPermission               = "access-control:permissions:read"
	PermissionsUpdatePermission             = "access-control:permissions:update"
	PermissionsDeletePermission             = "access-control:permissions:delete"
	RolePermissionsAssignPermission         = "access-control:role-permissions:assign"
	RolePermissionsReadPermission           = "access-control:role-permissions:read"
	RolePermissionsRemovePermission         = "access-control:role-permissions:remove"
	UserRolesAssignPermission               = "access-control:user-roles:assign"
	UserRolesReadPermission                 = "access-control:user-roles:read"
	UserRolesRemovePermission               = "access-control:user-roles:remove"
	UserPermissionsReadPermission           = "access-control:user-permissions:read"
	UserPermissionsCheckPermission          = "access-control:user-permissions:check"
)
