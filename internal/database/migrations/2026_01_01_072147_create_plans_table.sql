-- +migrate Up
CREATE TABLE "plans" (
  "plan_id" varchar(36) primary key,
  "name" varchar(100) not null unique,
  "external_id" text,
  "max_seats" integer not null,
  "min_seats" integer not null,
  "is_disabled" boolean not null default false,
  "version" varchar(20),
  "created_at" timestamp not null,
  "updated_at" timestamp not null,
  "deleted_at" text
);

-- +migrate Down
DROP TABLE plans;
