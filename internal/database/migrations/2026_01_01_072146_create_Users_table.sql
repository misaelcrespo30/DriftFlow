-- +migrate Up
CREATE TABLE "Users" (
  "user_id" varchar(36) primary key,
  "email" varchar(100) not null unique,
  "username" varchar(100) unique,
  "password_hash" varchar(250) not null,
  "access_failed_count" bigint not null default 0,
  "is_email_confirmed" boolean not null default false,
  "is_lockout_enabled" boolean not null default false,
  "lockout_end" timestamp,
  "phone" text,
  "is_phone_confirmed" boolean not null default false,
  "security_stamp" text,
  "misael" varchar(100) unique
);

-- +migrate Down
DROP TABLE Users;
