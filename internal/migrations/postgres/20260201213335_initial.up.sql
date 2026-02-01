-- create "users" table
CREATE TABLE "public"."users" (
  "id" character varying NOT NULL,
  "name" character varying NOT NULL,
  "email" character varying NOT NULL,
  "email_verified" boolean NOT NULL DEFAULT false,
  "image" character varying NULL,
  "metadata" json NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "users_email_key" UNIQUE ("email")
);
-- create "accounts" table
CREATE TABLE "public"."accounts" (
  "id" character varying NOT NULL,
  "user_id" character varying NOT NULL,
  "account_id" character varying NOT NULL,
  "provider_id" character varying NOT NULL,
  "access_token" character varying NULL,
  "refresh_token" character varying NULL,
  "id_token" character varying NULL,
  "access_token_expires_at" timestamptz NULL,
  "refresh_token_expires_at" timestamptz NULL,
  "scope" character varying NULL,
  "password" character varying NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "idx_accounts_provider_account" UNIQUE ("account_id", "provider_id"),
  CONSTRAINT "accounts_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create "sessions" table
CREATE TABLE "public"."sessions" (
  "id" character varying NOT NULL,
  "user_id" character varying NOT NULL,
  "token" character varying NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "ip_address" character varying NULL,
  "user_agent" character varying NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "sessions_token_key" UNIQUE ("token"),
  CONSTRAINT "sessions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create "verifications" table
CREATE TABLE "public"."verifications" (
  "id" character varying NOT NULL,
  "user_id" character varying NULL,
  "identifier" character varying NOT NULL,
  "token" character varying NOT NULL,
  "type" character varying NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "verifications_token_key" UNIQUE ("token"),
  CONSTRAINT "verifications_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
