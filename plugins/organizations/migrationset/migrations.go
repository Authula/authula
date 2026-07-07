package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func Migrations() []migrations.Migration {
	return []migrations.Migration{postgresInitial()}
}

func postgresInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260409000000_organizations_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE OR REPLACE FUNCTION organizations_set_updated_at_fn()
				RETURNS TRIGGER AS $$
					BEGIN
						NEW.updated_at = NOW();
						RETURN NEW;
					END;
				$$ LANGUAGE plpgsql;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organizations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					owner_id UUID NOT NULL,
					name VARCHAR(255) NOT NULL,
					slug VARCHAR(255) NOT NULL UNIQUE,
					logo TEXT,
					metadata JSONB,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT fk_organizations_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
				);`,
				`DROP TRIGGER IF EXISTS update_organizations_updated_at_trigger ON organizations;`,
				`CREATE TRIGGER update_organizations_updated_at_trigger
				BEFORE UPDATE ON organizations
				FOR EACH ROW
				EXECUTE FUNCTION organizations_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_organizations_owner_id ON organizations(owner_id);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_invitations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					organization_id UUID NOT NULL,
					inviter_id UUID NOT NULL,
					email VARCHAR(255) NOT NULL,
					role VARCHAR(255) NOT NULL,
					status VARCHAR(32) NOT NULL DEFAULT 'pending',
					expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT fk_organization_invitations_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					CONSTRAINT fk_organization_invitations_inviter FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE,
					CONSTRAINT chk_organization_invitations_status CHECK (status IN ('pending', 'accepted', 'rejected', 'revoked', 'expired'))
				);`,
				`DROP TRIGGER IF EXISTS update_organization_invitations_updated_at_trigger ON organization_invitations;`,
				`CREATE TRIGGER update_organization_invitations_updated_at_trigger
				BEFORE UPDATE ON organization_invitations
				FOR EACH ROW
				EXECUTE FUNCTION organizations_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_organization_id ON organization_invitations(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_inviter_id ON organization_invitations(inviter_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_email ON organization_invitations(email);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_status_expires_at ON organization_invitations(status, expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_members (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					organization_id UUID NOT NULL,
					user_id UUID NOT NULL,
					role VARCHAR(255) NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT fk_organization_members_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					CONSTRAINT fk_organization_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					CONSTRAINT uq_organization_members_organization_user UNIQUE (organization_id, user_id)
				);`,
				`DROP TRIGGER IF EXISTS update_organization_members_updated_at_trigger ON organization_members;`,
				`CREATE TRIGGER update_organization_members_updated_at_trigger
				BEFORE UPDATE ON organization_members
				FOR EACH ROW
				EXECUTE FUNCTION organizations_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_organization_members_organization_id ON organization_members(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_members_user_id ON organization_members(user_id);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_teams (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					organization_id UUID NOT NULL,
					name VARCHAR(255) NOT NULL,
					slug VARCHAR(255) NOT NULL,
					description TEXT,
					metadata JSONB,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT fk_organization_teams_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					CONSTRAINT uq_organization_teams_organization_slug UNIQUE (organization_id, slug)
				);`,
				`DROP TRIGGER IF EXISTS update_organization_teams_updated_at_trigger ON organization_teams;`,
				`CREATE TRIGGER update_organization_teams_updated_at_trigger
				BEFORE UPDATE ON organization_teams
				FOR EACH ROW
				EXECUTE FUNCTION organizations_set_updated_at_fn();`,
				`CREATE INDEX IF NOT EXISTS idx_organization_teams_organization_id ON organization_teams(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_teams_slug ON organization_teams(slug);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_team_members (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					team_id UUID NOT NULL,
					member_id UUID NOT NULL,
					created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
					CONSTRAINT fk_organization_team_members_team FOREIGN KEY (team_id) REFERENCES organization_teams(id) ON DELETE CASCADE,
					CONSTRAINT fk_organization_team_members_member FOREIGN KEY (member_id) REFERENCES organization_members(id) ON DELETE CASCADE,
					CONSTRAINT uq_organization_team_members_team_member UNIQUE (team_id, member_id)
				);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_team_members_team_id ON organization_team_members(team_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_team_members_member_id ON organization_team_members(member_id);`,
				// -----------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS organization_team_members;`,
				`DROP TRIGGER IF EXISTS update_organization_teams_updated_at_trigger ON organization_teams;`,
				`DROP TABLE IF EXISTS organization_teams;`,
				`DROP TRIGGER IF EXISTS update_organization_members_updated_at_trigger ON organization_members;`,
				`DROP TABLE IF EXISTS organization_members;`,
				`DROP TRIGGER IF EXISTS update_organization_invitations_updated_at_trigger ON organization_invitations;`,
				`DROP TABLE IF EXISTS organization_invitations;`,
				`DROP TRIGGER IF EXISTS update_organizations_updated_at_trigger ON organizations;`,
				`DROP TABLE IF EXISTS organizations;`,
				`DROP FUNCTION IF EXISTS organizations_set_updated_at_fn();`,
			)
		},
	}
}
