package models

import (
	"slices"
)

type ActorType string

const (
	ActorUser    ActorType = "user"
	ActorAPIKey  ActorType = "api_key"
	ActorMachine ActorType = "machine"
)

// Actor represents the fully resolved identity context of the caller.
type Actor struct {
	// Represents the unique identifier of the credential/actor.
	ID string

	// Indicates the type of actor
	Type ActorType

	// Holds the scopes granted to this specific request credential.
	Scopes []string

	// Flexible field to attach additional information about the actor.
	Claims map[string]any
}

func (actor *Actor) HasScopes(scope string) bool {
	if actor == nil || scope == "" {
		return false
	}
	return slices.Contains(actor.Scopes, scope)
}

func (actor *Actor) GetClaimString(key string) (string, bool) {
	if actor == nil || actor.Claims == nil {
		return "", false
	}
	val, ok := actor.Claims[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}
