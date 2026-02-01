-- create "key_value_store" table
CREATE TABLE "public"."key_value_store" (
  "key" character varying(255) NOT NULL,
  "value" character varying NULL,
  "expires_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("key")
);
