package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testActorID = "14643c3a-4e41-435c-9572-80cbb46c8a62"
)

func TestHasScopes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		actor      *Actor
		scope      string
		wantTruthy bool
	}{
		{
			name:       "returns false when there is no actor",
			actor:      nil,
			scope:      "test:scope",
			wantTruthy: false,
		},
		{
			name: "returns false when there is no scope specified",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Scopes: []string{"test:scope"},
			},
			scope:      "",
			wantTruthy: false,
		},
		{
			name: "returns false when the actor has no scopes",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Scopes: nil,
			},
			scope:      "test:scope",
			wantTruthy: false,
		},
		{
			name: "returns false when the actor doesn't have the scope",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Scopes: []string{"other:scope"},
			},
			scope:      "test:scope",
			wantTruthy: false,
		},
		{
			name: "returns false for a scope that is only a prefix of a granted scope",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Scopes: []string{"test:scope"},
			},
			scope:      "test",
			wantTruthy: false,
		},
		{
			name: "returns true when the actor has the scope",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Scopes: []string{"test:scope"},
			},
			scope:      "test:scope",
			wantTruthy: true,
		},
		{
			name: "returns true when the actor has the scope among others",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorMachine,
				Scopes: []string{"other:scope", "test:scope", "another:scope"},
			},
			scope:      "test:scope",
			wantTruthy: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.actor.HasScopes(tt.scope)

			assert.Equal(t, tt.wantTruthy, got)
		})
	}
}

func TestGetClaimString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		actor     *Actor
		key       string
		wantValue string
		wantOK    bool
	}{
		{
			name:      "returns false when there is no actor",
			actor:     nil,
			key:       "role",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "returns false when the actor has no claims",
			actor: &Actor{
				ID:   testActorID,
				Type: ActorUser,
			},
			key:       "role",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "returns false when the claim is missing",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Claims: map[string]any{"role": "admin"},
			},
			key:       "email",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "returns false when the key is empty",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Claims: map[string]any{"role": "admin"},
			},
			key:       "",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "returns false when the claim is not a string",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Claims: map[string]any{"role": 42},
			},
			key:       "role",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "returns false when the claim is nil",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Claims: map[string]any{"role": nil},
			},
			key:       "role",
			wantValue: "",
			wantOK:    false,
		},
		{
			name: "returns the value when the claim is a string",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Claims: map[string]any{"role": "admin"},
			},
			key:       "role",
			wantValue: "admin",
			wantOK:    true,
		},
		{
			name: "returns an empty value for an empty string claim",
			actor: &Actor{
				ID:     testActorID,
				Type:   ActorUser,
				Claims: map[string]any{"role": ""},
			},
			key:       "role",
			wantValue: "",
			wantOK:    true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotValue, gotOK := tt.actor.GetClaimString(tt.key)

			assert.Equal(t, tt.wantValue, gotValue)
			assert.Equal(t, tt.wantOK, gotOK)
		})
	}
}
