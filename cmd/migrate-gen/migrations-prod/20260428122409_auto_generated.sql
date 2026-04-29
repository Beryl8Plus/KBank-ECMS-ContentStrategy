-- +goose Up
-- Auto-generated from GORM entity changes on 2026-04-28T12:24:09+07:00
-- Source: atlas schema diff (previous snapshot → current GORM state)

-- Add new schema named "public"
CREATE SCHEMA IF NOT EXISTS "public";
-- Set comment to schema: "public"
COMMENT ON SCHEMA "public" IS 'standard public schema';
-- Create "calendars" table
CREATE TABLE "public"."calendars" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "NAME" character varying(255) NULL,
  "TYPE" character varying(255) NULL,
  "IS_ACTIVE" boolean NULL DEFAULT true,
  PRIMARY KEY ("ID")
);
-- Create index "idx_calendars_deleted_at" to table: "calendars"
CREATE INDEX "idx_calendars_deleted_at" ON "public"."calendars" ("DELETED_AT");
-- Create "clen_schema_registry" table
CREATE TABLE "public"."clen_schema_registry" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "SCHEMA_NAME" character varying(255) NULL,
  "VERSION" character varying(255) NULL,
  "SCHEMA_DEFINITION" jsonb NULL,
  "IS_ACTIVE" boolean NULL DEFAULT false,
  PRIMARY KEY ("ID")
);
-- Create index "idx_clen_schema_registry_deleted_at" to table: "clen_schema_registry"
CREATE INDEX "idx_clen_schema_registry_deleted_at" ON "public"."clen_schema_registry" ("DELETED_AT");
-- Create "decision_rule_id_sequences" table
CREATE TABLE "public"."decision_rule_id_sequences" (
  "year_month" text NOT NULL,
  "last_seq" integer NOT NULL DEFAULT 0,
  CONSTRAINT "pk_decision_rule_id_seq" PRIMARY KEY ("year_month")
);
-- Create "login_token_histories" table
CREATE TABLE "public"."login_token_histories" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "USER_NAME" character varying(255) NULL,
  "ACCESS_TOKEN" character varying(255) NULL,
  "EXPIRE_DATE" timestamptz NULL,
  PRIMARY KEY ("ID")
);
-- Create index "idx_login_token_histories_deleted_at" to table: "login_token_histories"
CREATE INDEX "idx_login_token_histories_deleted_at" ON "public"."login_token_histories" ("DELETED_AT");
-- Create index "idx_login_token_histories_user_name" to table: "login_token_histories"
CREATE UNIQUE INDEX "idx_login_token_histories_user_name" ON "public"."login_token_histories" ("USER_NAME");
-- Create "calendar_dates" table
CREATE TABLE "public"."calendar_dates" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "CALENDAR_ID" uuid NOT NULL,
  "DATE" date NULL,
  "NAME" character varying(255) NULL,
  "IS_RECURRING" boolean NULL DEFAULT false,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_calendar_dates_calendar" FOREIGN KEY ("CALENDAR_ID") REFERENCES "public"."calendars" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_calendar_dates_deleted_at" to table: "calendar_dates"
CREATE INDEX "idx_calendar_dates_deleted_at" ON "public"."calendar_dates" ("DELETED_AT");
-- Create "profiles" table
CREATE TABLE "public"."profiles" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "NAME" character varying(255) NULL,
  "CODE" character varying(255) NULL,
  PRIMARY KEY ("ID")
);
-- Create index "idx_profiles_code" to table: "profiles"
CREATE UNIQUE INDEX "idx_profiles_code" ON "public"."profiles" ("CODE");
-- Create index "idx_profiles_deleted_at" to table: "profiles"
CREATE INDEX "idx_profiles_deleted_at" ON "public"."profiles" ("DELETED_AT");
-- Create "roles" table
CREATE TABLE "public"."roles" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "NAME" character varying(255) NULL,
  "CODE" character varying(255) NULL,
  PRIMARY KEY ("ID")
);
-- Create index "idx_roles_code" to table: "roles"
CREATE UNIQUE INDEX "idx_roles_code" ON "public"."roles" ("CODE");
-- Create index "idx_roles_deleted_at" to table: "roles"
CREATE INDEX "idx_roles_deleted_at" ON "public"."roles" ("DELETED_AT");
-- Create "users" table
CREATE TABLE "public"."users" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "ROLE_ID" uuid NULL,
  "PROFILE_ID" uuid NULL,
  "EMAIL" character varying(255) NULL,
  "NAME_TH" character varying(255) NULL,
  "NAME_EN" character varying(255) NULL,
  "IS_ACTIVE" boolean NULL DEFAULT true,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_users_profile" FOREIGN KEY ("PROFILE_ID") REFERENCES "public"."profiles" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_users_role" FOREIGN KEY ("ROLE_ID") REFERENCES "public"."roles" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "public"."users" ("DELETED_AT");
-- Create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "public"."users" ("EMAIL");
-- Create "decision_rules" table
CREATE TABLE "public"."decision_rules" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "DECISION_RULE_RUNNING" character varying(255) NOT NULL,
  "NAME" character varying(255) NOT NULL,
  "TYPE" character varying(255) NOT NULL,
  "EVALUATE_TYPE" character varying(255) NOT NULL,
  "CONTENT_PATH" character varying(255) NOT NULL,
  "CAMPAIGN_CODE" character varying(25) NULL,
  "SCORE" numeric NULL DEFAULT 0,
  "STATUS" character varying(255) NULL,
  "SUB_STATUS" character varying(255) NULL,
  "INACTIVE_BY" uuid NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_decision_rules_inactive_by_user" FOREIGN KEY ("INACTIVE_BY") REFERENCES "public"."users" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_decision_rules_active_status" to table: "decision_rules"
CREATE INDEX "idx_decision_rules_active_status" ON "public"."decision_rules" ("STATUS") WHERE ((("STATUS")::text = 'ACTIVE'::text) AND ("DELETED_AT" IS NULL));
-- Create index "idx_decision_rules_decision_rule_running" to table: "decision_rules"
CREATE UNIQUE INDEX "idx_decision_rules_decision_rule_running" ON "public"."decision_rules" ("DECISION_RULE_RUNNING");
-- Create index "idx_decision_rules_deleted_at" to table: "decision_rules"
CREATE INDEX "idx_decision_rules_deleted_at" ON "public"."decision_rules" ("DELETED_AT");
-- Create "channels" table
CREATE TABLE "public"."channels" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "CHANNEL_NAME" character varying(255) NULL,
  PRIMARY KEY ("ID")
);
-- Create index "idx_channels_deleted_at" to table: "channels"
CREATE INDEX "idx_channels_deleted_at" ON "public"."channels" ("DELETED_AT");
-- Create "placements" table
CREATE TABLE "public"."placements" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "PLACEMENT_NAME" character varying(255) NOT NULL,
  "CHANNEL_ID" uuid NOT NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_placements_channel" FOREIGN KEY ("CHANNEL_ID") REFERENCES "public"."channels" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_channel_placement_name" to table: "placements"
CREATE UNIQUE INDEX "idx_channel_placement_name" ON "public"."placements" ("PLACEMENT_NAME", "CHANNEL_ID");
-- Create index "idx_placements_deleted_at" to table: "placements"
CREATE INDEX "idx_placements_deleted_at" ON "public"."placements" ("DELETED_AT");
-- Create "permissions" table
CREATE TABLE "public"."permissions" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "NAME" character varying(255) NULL,
  "PERMISSION_TYPE" character varying(255) NULL,
  "SOURCE" character varying(255) NULL,
  "ACTION" character varying(255) NULL,
  PRIMARY KEY ("ID")
);
-- Create index "idx_permissions_deleted_at" to table: "permissions"
CREATE INDEX "idx_permissions_deleted_at" ON "public"."permissions" ("DELETED_AT");
-- Create index "idx_source_action" to table: "permissions"
CREATE UNIQUE INDEX "idx_source_action" ON "public"."permissions" ("SOURCE", "ACTION");
-- Create "profile_permissions" table
CREATE TABLE "public"."profile_permissions" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "PROFILE_ID" uuid NULL,
  "PERMISSION_ID" uuid NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_profile_permissions_permission" FOREIGN KEY ("PERMISSION_ID") REFERENCES "public"."permissions" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_profile_permissions_profile" FOREIGN KEY ("PROFILE_ID") REFERENCES "public"."profiles" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_profile_permission" to table: "profile_permissions"
CREATE UNIQUE INDEX "idx_profile_permission" ON "public"."profile_permissions" ("PROFILE_ID", "PERMISSION_ID");
-- Create index "idx_profile_permissions_deleted_at" to table: "profile_permissions"
CREATE INDEX "idx_profile_permissions_deleted_at" ON "public"."profile_permissions" ("DELETED_AT");
-- Create "attributes" table
CREATE TABLE "public"."attributes" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "CLEN_SCHEMA_REGISTRY_ID" uuid NOT NULL,
  "FIELD_NAME" character varying(255) NULL,
  "DISPLAY_NAME" character varying(255) NULL,
  "DATA_TYPE" character varying(255) NULL,
  "VALUE" jsonb NULL,
  "DESCRIPTION" text NULL,
  "SOURCE_SYSTEM" character varying(255) NULL,
  "IS_ACTIVE" boolean NULL DEFAULT true,
  PRIMARY KEY ("ID")
);
-- Create index "idx_attributes_deleted_at" to table: "attributes"
CREATE INDEX "idx_attributes_deleted_at" ON "public"."attributes" ("DELETED_AT");
-- Create "rules" table
CREATE TABLE "public"."rules" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "DECISION_RULE_ID" uuid NOT NULL,
  "VARIATION_NAME" character varying(255) NULL,
  "SCORE" numeric NULL,
  "ORDER_NO" integer NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_decision_rules_rules" FOREIGN KEY ("DECISION_RULE_ID") REFERENCES "public"."decision_rules" ("ID") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create index "idx_rules_deleted_at" to table: "rules"
CREATE INDEX "idx_rules_deleted_at" ON "public"."rules" ("DELETED_AT");
-- Create "rule_attributes" table
CREATE TABLE "public"."rule_attributes" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "RULE_ID" uuid NOT NULL,
  "ATTRIBUTE_ID" uuid NOT NULL,
  "VALUE" jsonb NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_rule_attributes_attribute" FOREIGN KEY ("ATTRIBUTE_ID") REFERENCES "public"."attributes" ("ID") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_rules_rule_attributes" FOREIGN KEY ("RULE_ID") REFERENCES "public"."rules" ("ID") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create index "idx_rule_attributes_deleted_at" to table: "rule_attributes"
CREATE INDEX "idx_rule_attributes_deleted_at" ON "public"."rule_attributes" ("DELETED_AT");
-- Create "rule_conditions" table
CREATE TABLE "public"."rule_conditions" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "SEQUENCE" integer NULL,
  "DECISION_RULE_ID" uuid NOT NULL,
  "PARENT_RULE_CONDITION_ID" uuid NULL,
  "ATTRIBUTE_ID" uuid NOT NULL,
  "LOGICAL_OPERATOR" character varying(50) NULL,
  "CONNECTOR_OPERATOR" character varying(50) NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_decision_rules_rule_conditions" FOREIGN KEY ("DECISION_RULE_ID") REFERENCES "public"."decision_rules" ("ID") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_rule_conditions_attribute" FOREIGN KEY ("ATTRIBUTE_ID") REFERENCES "public"."attributes" ("ID") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "fk_rule_conditions_rule_condition_children" FOREIGN KEY ("PARENT_RULE_CONDITION_ID") REFERENCES "public"."rule_conditions" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_rule_conditions_deleted_at" to table: "rule_conditions"
CREATE INDEX "idx_rule_conditions_deleted_at" ON "public"."rule_conditions" ("DELETED_AT");
-- Create "schedules" table
CREATE TABLE "public"."schedules" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "DECISION_RULE_ID" uuid NULL,
  "PLACEMENT_ID" uuid NULL,
  "CALENDAR_ID" uuid NULL,
  "RECURRENCE_TYPE" character varying(255) NULL,
  "RECURRENCE_RULE" text NULL,
  "EFFECTIVE_FROM" timestamptz NULL,
  "EFFECTIVE_UNTIL" timestamptz NULL,
  "TIME_OF_DAY_START" character varying(5) NULL,
  "TIME_OF_DAY_END" character varying(5) NULL,
  "ALL_DAY" boolean NULL DEFAULT false,
  "TIMEZONE" character varying(255) NULL DEFAULT 'Asia/Bangkok',
  "IS_ACTIVE" boolean NULL DEFAULT false,
  PRIMARY KEY ("ID"),
  CONSTRAINT "no_overlap_active_schedule_per_rule_placement" EXCLUDE USING GIST ("DECISION_RULE_ID" WITH =, "PLACEMENT_ID" WITH =, (tstzrange("EFFECTIVE_FROM", "EFFECTIVE_UNTIL")) WITH &&) WHERE (("IS_ACTIVE" = true) AND ("DELETED_AT" IS NULL)),
  CONSTRAINT "fk_schedules_calendar" FOREIGN KEY ("CALENDAR_ID") REFERENCES "public"."calendars" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_schedules_decision_rule" FOREIGN KEY ("DECISION_RULE_ID") REFERENCES "public"."decision_rules" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_schedules_placement" FOREIGN KEY ("PLACEMENT_ID") REFERENCES "public"."placements" ("ID") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_schedules_active_window" to table: "schedules"
CREATE INDEX "idx_schedules_active_window" ON "public"."schedules" ("EFFECTIVE_FROM", "EFFECTIVE_UNTIL") WHERE (("IS_ACTIVE" = true) AND ("DELETED_AT" IS NULL));
-- Create index "idx_schedules_created_at_desc_active" to table: "schedules"
CREATE INDEX "idx_schedules_created_at_desc_active" ON "public"."schedules" ("CREATED_AT" DESC) WHERE ("DELETED_AT" IS NULL);
-- Create index "idx_schedules_deleted_at" to table: "schedules"
CREATE INDEX "idx_schedules_deleted_at" ON "public"."schedules" ("DELETED_AT");
-- Create "schedule_occurrences" table
CREATE TABLE "public"."schedule_occurrences" (
  "ID" uuid NOT NULL DEFAULT gen_random_uuid(),
  "CREATED_AT" timestamptz NULL,
  "CREATED_BY" uuid NULL,
  "UPDATED_AT" timestamptz NULL,
  "UPDATED_BY" uuid NULL,
  "DELETED_AT" timestamptz NULL,
  "SCHEDULE_ID" uuid NOT NULL,
  "OCCURRENCE_START" timestamptz NULL,
  "OCCURRENCE_END" timestamptz NULL,
  "STATUS" character varying(255) NULL,
  "SOURCE" character varying(255) NULL,
  PRIMARY KEY ("ID"),
  CONSTRAINT "fk_schedule_occurrences_schedule" FOREIGN KEY ("SCHEDULE_ID") REFERENCES "public"."schedules" ("ID") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create index "idx_occurrence_schedule_start_end" to table: "schedule_occurrences"
CREATE UNIQUE INDEX "idx_occurrence_schedule_start_end" ON "public"."schedule_occurrences" ("SCHEDULE_ID", "OCCURRENCE_START", "OCCURRENCE_END");
-- Create index "idx_schedule_occurrences_deleted_at" to table: "schedule_occurrences"
CREATE INDEX "idx_schedule_occurrences_deleted_at" ON "public"."schedule_occurrences" ("DELETED_AT");

-- +goose Down
-- Rollback for auto-generated migration

DROP TABLE IF EXISTS "users" CASCADE;
DROP TABLE IF EXISTS "schedules" CASCADE;
DROP TABLE IF EXISTS "schedule_occurrences" CASCADE;
DROP TABLE IF EXISTS "rules" CASCADE;
DROP TABLE IF EXISTS "rule_conditions" CASCADE;
DROP TABLE IF EXISTS "rule_attributes" CASCADE;
DROP TABLE IF EXISTS "roles" CASCADE;
DROP TABLE IF EXISTS "profiles" CASCADE;
DROP TABLE IF EXISTS "profile_permissions" CASCADE;
DROP TABLE IF EXISTS "placements" CASCADE;
DROP TABLE IF EXISTS "permissions" CASCADE;
DROP TABLE IF EXISTS "login_token_histories" CASCADE;
DROP TABLE IF EXISTS "decision_rules" CASCADE;
DROP TABLE IF EXISTS "decision_rule_id_sequences" CASCADE;
DROP TABLE IF EXISTS "clen_schema_registry" CASCADE;
DROP TABLE IF EXISTS "channels" CASCADE;
DROP TABLE IF EXISTS "calendars" CASCADE;
DROP TABLE IF EXISTS "calendar_dates" CASCADE;
DROP TABLE IF EXISTS "attributes" CASCADE;
