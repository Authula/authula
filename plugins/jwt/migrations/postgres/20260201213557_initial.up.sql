-- create "jwks" table
CREATE TABLE "public"."jwks" (
  "id" character varying NOT NULL,
  "public_key" character varying NOT NULL,
  "private_key" character varying NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "expires_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create "refresh_tokens" table
CREATE TABLE "public"."refresh_tokens" (
  "id" character varying NOT NULL,
  "session_id" character varying NOT NULL,
  "token_hash" character varying NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "is_revoked" boolean NOT NULL DEFAULT false,
  "revoked_at" timestamptz NULL,
  "last_reuse_attempt" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "refresh_tokens_token_hash_key" UNIQUE ("token_hash")
);
