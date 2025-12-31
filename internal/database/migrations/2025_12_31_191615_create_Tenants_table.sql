-- +migrate Up
CREATE TABLE "Tenants" (
  "tenant_id" varchar(36) primary key,
  "tenant_name" varchar(200) not null,
  "service_plan" varchar(50) not null default 'standard',
  "recovery_state" varchar(50) not null,
  "last_updated" timestamp not null,
  "connection_string" text not null,
  "is_available" boolean not null default false,
  "is_data_seeded" boolean not null default false,
  "should_delete" boolean not null default false,
  "firm_name" varchar(200),
  "quantity_of_employees" bigint not null default 0,
  "plan_id" text,
  "seats_allowed" bigint not null default 0,
  "seats_used" bigint not null default 0,
  "subscription_customer_id" text,
  "subscription_id" text,
  "subscription_item_id" text,
  "misael" varchar(50) not null default 'standard',
  "crespo" varchar(100) unique,
  FOREIGN KEY ("plan_id") REFERENCES "plans"("plan_id")
);

-- +migrate Down
DROP TABLE Tenants;
