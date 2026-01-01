-- +migrate Up
CREATE TABLE "TenantUsers" (
  "id" varchar(36) primary key,
  "user_id" varchar(36) not null,
  "tenant_id" varchar(36) not null,
  "relationship" bigint not null,
  "is_active" boolean not null default false,
  "is_default" boolean not null default false,
  "originated_user" boolean not null default false,
  "external_identity_id" varchar(100) not null,
  FOREIGN KEY ("user_id") REFERENCES "Users"("user_id"),
  FOREIGN KEY ("tenant_id") REFERENCES "Tenants"("tenant_id")
);

-- +migrate Down
DROP TABLE TenantUsers;
