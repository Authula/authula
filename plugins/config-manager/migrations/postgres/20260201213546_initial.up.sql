-- create "auth_settings" table
CREATE TABLE "public"."auth_settings" (
  "config_version" bigserial NOT NULL,
  "key" character varying(255) NULL,
  "value" json NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("config_version"),
  CONSTRAINT "auth_settings_key_key" UNIQUE ("key")
);
