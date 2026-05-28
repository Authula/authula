package models

import (
	"context"
	"net/http"
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

// SetActorInContext updates the actor in both the RequestContext and the underlying Go context stream.
func (reqCtx *RequestContext) SetActorInContext(actor *Actor) {
	reqCtx.Actor = actor
	if reqCtx.Actor.Permissions == nil {
		reqCtx.Actor.Permissions = make([]string, 0)
	}
	if reqCtx.Actor.Metadata == nil {
		reqCtx.Actor.Metadata = make(map[string]any)
	}
	reqCtx.Request = reqCtx.Request.WithContext(context.WithValue(reqCtx.Request.Context(), ContextAuthActor, actor))
}

// GetActorFromContext extracts the actor from the Go context or RequestContext fallback.
func GetActorFromContext(ctx context.Context) (*Actor, bool) {
	if val := ctx.Value(ContextAuthActor); val != nil {
		if actor, ok := val.(*Actor); ok {
			return actor, ok
		}
	}

	if reqCtx, ok := GetRequestContext(ctx); ok && reqCtx.Actor != nil {
		return reqCtx.Actor, true
	}

	return nil, false
}

// GetActorFromRequest extracts the actor directly from an incoming HTTP request.
func GetActorFromRequest(req *http.Request) (*Actor, bool) {
	return GetActorFromContext(req.Context())
}
