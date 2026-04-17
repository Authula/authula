package migrationset

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/migrations"
)

func ForProvider(provider string) []migrations.Migration {
	return migrations.ForProvider(provider, migrations.ProviderVariants{
		"sqlite":   func() []migrations.Migration { return []migrations.Migration{organizationsSQLiteInitial()} },
		"postgres": func() []migrations.Migration { return []migrations.Migration{organizationsPostgresInitial()} },
		"mysql":    func() []migrations.Migration { return []migrations.Migration{organizationsMySQLInitial()} },
	})
}

func organizationsSQLiteInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260409000000_organizations_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`PRAGMA foreign_keys = ON;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organizations (
					id TEXT PRIMARY KEY,
					owner_id TEXT NOT NULL,
					name VARCHAR(255) NOT NULL,
					slug VARCHAR(255) NOT NULL UNIQUE,
					logo TEXT,
					metadata TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
				);`,
				`CREATE INDEX IF NOT EXISTS idx_organizations_owner_id ON organizations(owner_id);`,
				`DROP TRIGGER IF EXISTS update_organizations_updated_at_trigger;`,
				`CREATE TRIGGER update_organizations_updated_at_trigger 
				AFTER UPDATE ON organizations
				FOR EACH ROW
				WHEN NEW.updated_at <= OLD.updated_at
				BEGIN
				  UPDATE organizations SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
				END;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_invitations (
					id TEXT PRIMARY KEY,
					email VARCHAR(255) NOT NULL,
					inviter_id TEXT NOT NULL,
					organization_id TEXT NOT NULL,
					role VARCHAR(255) NOT NULL,
					status VARCHAR(32) NOT NULL DEFAULT 'pending',
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE
				);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_email ON organization_invitations(email);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_organization_id ON organization_invitations(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_inviter_id ON organization_invitations(inviter_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_invitations_status_expires_at ON organization_invitations(status, expires_at);`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_members (
					id TEXT PRIMARY KEY,
					organization_id TEXT NOT NULL,
					user_id TEXT NOT NULL,
					role VARCHAR(255) NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					UNIQUE (organization_id, user_id)
				);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_members_organization_id ON organization_members(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_members_user_id ON organization_members(user_id);`,
				`DROP TRIGGER IF EXISTS update_organization_members_updated_at_trigger;`,
				`CREATE TRIGGER update_organization_members_updated_at_trigger 
        AFTER UPDATE ON organization_members
        BEGIN
          UPDATE organization_members SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_teams (
					id TEXT PRIMARY KEY,
					organization_id TEXT NOT NULL,
					name VARCHAR(255) NOT NULL UNIQUE,
					slug VARCHAR(255) NOT NULL UNIQUE,
					description TEXT,
					metadata TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					UNIQUE (organization_id, slug)
				);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_teams_organization_id ON organization_teams(organization_id);`,
				`CREATE INDEX IF NOT EXISTS idx_organization_teams_slug ON organization_teams(slug);`,
				`DROP TRIGGER IF EXISTS update_organization_teams_updated_at_trigger;`,
				`CREATE TRIGGER update_organization_teams_updated_at_trigger 
        AFTER UPDATE ON organization_teams
        BEGIN
          UPDATE organization_teams SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_team_members (
					id TEXT PRIMARY KEY,
					team_id TEXT NOT NULL,
					member_id TEXT NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (team_id) REFERENCES organization_teams(id) ON DELETE CASCADE,
					FOREIGN KEY (member_id) REFERENCES organization_members(id) ON DELETE CASCADE,
					UNIQUE (team_id, member_id)
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
				`DROP TABLE IF EXISTS organization_teams;`,
				`DROP TABLE IF EXISTS organization_members;`,
				`DROP TABLE IF EXISTS organization_invitations;`,
				`DROP TABLE IF EXISTS organizations;`,
			)
		},
	}
}

func organizationsPostgresInitial() migrations.Migration {
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

func organizationsMySQLInitial() migrations.Migration {
	return migrations.Migration{
		Version: "20260409000000_organizations_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organizations (
					id BINARY(16) NOT NULL PRIMARY KEY,
					owner_id BINARY(16) NOT NULL,
					name VARCHAR(255) NOT NULL,
					slug VARCHAR(255) NOT NULL UNIQUE,
					logo TEXT NULL,
					metadata JSON NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT fk_organizations_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
					INDEX idx_organizations_owner_id (owner_id)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_invitations (
					id BINARY(16) NOT NULL PRIMARY KEY,
					organization_id BINARY(16) NOT NULL,
					inviter_id BINARY(16) NOT NULL,
					email VARCHAR(255) NOT NULL,
					role VARCHAR(255) NOT NULL,
					status VARCHAR(32) NOT NULL DEFAULT 'pending',
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT fk_organization_invitations_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					CONSTRAINT fk_organization_invitations_inviter FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE,
					CONSTRAINT chk_organization_invitations_status CHECK (status IN ('pending', 'accepted', 'rejected', 'revoked', 'expired')),
					INDEX idx_organization_invitations_organization_id (organization_id),
					INDEX idx_organization_invitations_inviter_id (inviter_id),
					INDEX idx_organization_invitations_email (email),
					INDEX idx_organization_invitations_status_expires_at (status, expires_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_members (
					id BINARY(16) NOT NULL PRIMARY KEY,
					organization_id BINARY(16) NOT NULL,
					user_id BINARY(16) NOT NULL,
					role VARCHAR(255) NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT fk_organization_members_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					CONSTRAINT fk_organization_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					CONSTRAINT uq_organization_members_organization_user UNIQUE (organization_id, user_id),
					INDEX idx_organization_members_organization_id (organization_id),
					INDEX idx_organization_members_user_id (user_id)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_teams (
					id BINARY(16) NOT NULL PRIMARY KEY,
					organization_id BINARY(16) NOT NULL,
					name VARCHAR(255) NOT NULL,
					slug VARCHAR(255) NOT NULL,
					description TEXT NULL,
					metadata JSON NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					CONSTRAINT fk_organization_teams_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
					CONSTRAINT uq_organization_teams_organization_slug UNIQUE (organization_id, slug),
					INDEX idx_organization_teams_organization_id (organization_id),
					INDEX idx_organization_teams_slug (slug)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
				`CREATE TABLE IF NOT EXISTS organization_team_members (
					id BINARY(16) NOT NULL PRIMARY KEY,
					team_id BINARY(16) NOT NULL,
					member_id BINARY(16) NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT fk_organization_team_members_team FOREIGN KEY (team_id) REFERENCES organization_teams(id) ON DELETE CASCADE,
					CONSTRAINT fk_organization_team_members_member FOREIGN KEY (member_id) REFERENCES organization_members(id) ON DELETE CASCADE,
					CONSTRAINT uq_organization_team_members_team_member UNIQUE (team_id, member_id),
					INDEX idx_organization_team_members_team_id (team_id),
					INDEX idx_organization_team_members_member_id (member_id)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
				// -----------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS organization_team_members;`,
				`DROP TABLE IF EXISTS organization_teams;`,
				`DROP TABLE IF EXISTS organization_members;`,
				`DROP TABLE IF EXISTS organization_invitations;`,
				`DROP TABLE IF EXISTS organizations;`,
			)
		},
	}
}
