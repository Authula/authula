-- +goose Up

-- Create a function to automatically update updated_at timestamp
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION config_manager_update_updated_at_column_func()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Create sequence for config versioning
CREATE SEQUENCE IF NOT EXISTS config_version_seq START WITH 1 INCREMENT BY 1;

CREATE TABLE IF NOT EXISTS auth_settings (
  config_version BIGINT PRIMARY KEY DEFAULT nextval('config_version_seq'),
  key VARCHAR(255) UNIQUE NOT NULL,
  value JSONB NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index for efficient polling by config_version
CREATE INDEX IF NOT EXISTS idx_auth_settings_config_version ON auth_settings(config_version);

DROP TRIGGER IF EXISTS update_auth_settings_updated_at_trigger ON auth_settings;
CREATE TRIGGER update_auth_settings_updated_at_trigger
  BEFORE UPDATE ON auth_settings
  FOR EACH ROW
  EXECUTE FUNCTION config_manager_update_updated_at_column_func();

-- +goose Down

DROP TRIGGER IF EXISTS update_auth_settings_updated_at_trigger ON auth_settings;
DROP TABLE IF EXISTS auth_settings;
DROP SEQUENCE IF EXISTS config_version_seq;
