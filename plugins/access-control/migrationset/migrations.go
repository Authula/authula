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
		Version: "20260309000000_access_control_initial",
		Up: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`CREATE OR REPLACE FUNCTION access_control_set_updated_at_fn()
        RETURNS TRIGGER AS $$
          BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
          END;
        $$ LANGUAGE plpgsql;`,
				// -------------------------------
				`CREATE TABLE IF NOT EXISTS access_control_roles (
          id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
          name VARCHAR(255) NOT NULL UNIQUE,
          description TEXT,
          weight INTEGER NOT NULL,
          is_system BOOLEAN NOT NULL DEFAULT FALSE,
          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
        );`,
				`DROP TRIGGER IF EXISTS update_access_control_roles_updated_at_trigger ON access_control_roles;`,
				`CREATE TRIGGER update_access_control_roles_updated_at_trigger
        BEFORE UPDATE ON access_control_roles
        FOR EACH ROW
        EXECUTE FUNCTION access_control_set_updated_at_fn();`,
				// -------------------------------
				`CREATE TABLE IF NOT EXISTS access_control_permissions (
          id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
          key VARCHAR(255) NOT NULL UNIQUE,
          description TEXT,
          is_system BOOLEAN NOT NULL DEFAULT FALSE,
          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
        );`,
				`DROP TRIGGER IF EXISTS update_access_control_permissions_updated_at_trigger ON access_control_permissions;`,
				`CREATE TRIGGER update_access_control_permissions_updated_at_trigger
        BEFORE UPDATE ON access_control_permissions
        FOR EACH ROW
        EXECUTE FUNCTION access_control_set_updated_at_fn();`,
				// -------------------------------
				`CREATE TABLE IF NOT EXISTS access_control_role_permissions (
          role_id UUID NOT NULL,
          permission_id UUID NOT NULL,
          granted_by_user_id UUID,
          granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          PRIMARY KEY (role_id, permission_id),
          CONSTRAINT fk_access_control_role_permissions_role FOREIGN KEY (role_id) REFERENCES access_control_roles(id) ON DELETE CASCADE,
          CONSTRAINT fk_access_control_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES access_control_permissions(id) ON DELETE CASCADE,
          CONSTRAINT fk_access_control_role_permissions_granted_by FOREIGN KEY (granted_by_user_id) REFERENCES users(id) ON DELETE SET NULL
        );`,
				`CREATE INDEX IF NOT EXISTS idx_access_control_role_permissions_role_id ON access_control_role_permissions(role_id);`,
				`CREATE INDEX IF NOT EXISTS idx_access_control_role_permissions_permission_id ON access_control_role_permissions(permission_id);`,
				// -------------------------------
				`CREATE TABLE IF NOT EXISTS access_control_user_roles (
          user_id UUID NOT NULL,
          role_id UUID NOT NULL,
          assigned_by_user_id UUID,
          assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
          expires_at TIMESTAMP WITH TIME ZONE,
          PRIMARY KEY (user_id, role_id),
          CONSTRAINT fk_access_control_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
          CONSTRAINT fk_access_control_user_roles_role FOREIGN KEY (role_id) REFERENCES access_control_roles(id) ON DELETE CASCADE,
          CONSTRAINT fk_access_control_user_roles_assigned_by FOREIGN KEY (assigned_by_user_id) REFERENCES users(id) ON DELETE SET NULL
        );`,
				`CREATE INDEX IF NOT EXISTS idx_access_control_user_roles_role_id ON access_control_user_roles(role_id);`,
				`CREATE INDEX IF NOT EXISTS idx_access_control_user_roles_expires_at ON access_control_user_roles(expires_at);`,
				// -------------------------------
			)
		},
		Down: func(ctx context.Context, tx bun.Tx) error {
			return migrations.ExecStatements(
				ctx,
				tx,
				`DROP TABLE IF EXISTS access_control_user_roles;`,
				`DROP TABLE IF EXISTS access_control_role_permissions;`,
				`DROP TRIGGER IF EXISTS update_access_control_roles_updated_at_trigger ON access_control_roles;`,
				`DROP TRIGGER IF EXISTS update_access_control_permissions_updated_at_trigger ON access_control_permissions;`,
				`DROP TABLE IF EXISTS access_control_permissions;`,
				`DROP TABLE IF EXISTS access_control_roles;`,
				`DROP FUNCTION IF EXISTS access_control_set_updated_at_fn();`,
			)
		},
	}
}
