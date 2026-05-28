package models

import (
	"slices"
)

type ActorType string

const (
	ActorUser    ActorType = "user"
	ActorMachine ActorType = "machine"
)

// Actor represents the fully resolved identity context of the caller.
type Actor struct {
	// Represents the unique identifier of the credential/actor.
	ID string

	// Indicates the type of actor
	Type ActorType

	// Identifies the tenant a user is actively operating inside an org.
	// This can be nil if a user is operating in a personal/non-tenant scope.
	OrganizationID *string

	// Holds the static permissions granted to this specific request credential.
	Permissions []string

	// Flexible field to attach additional information about the actor.
	Metadata map[string]any
}

func (actor *Actor) HasPermission(permission string) bool {
	if actor == nil || permission == "" {
		return false
	}

	return slices.Contains(actor.Permissions, permission)
}
