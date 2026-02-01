-- create "rate_limits" table
CREATE TABLE "public"."rate_limits" (
  "key" character varying NOT NULL,
  "count" bigint NULL,
  "expires_at" timestamptz NULL,
  PRIMARY KEY ("key")
);
