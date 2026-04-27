-- Modify "calendar_dates" table
ALTER TABLE "calendar_dates" DROP CONSTRAINT "fk_calendar_dates_calendar";
-- Modify "decision_rules" table
ALTER TABLE "decision_rules" DROP CONSTRAINT "fk_decision_rules_inactive_by_user";
-- Modify "placements" table
ALTER TABLE "placements" DROP CONSTRAINT "fk_placements_channel";
-- Modify "profile_permissions" table
ALTER TABLE "profile_permissions" DROP CONSTRAINT "fk_profile_permissions_permission", DROP CONSTRAINT "fk_profile_permissions_profile";
-- Modify "rule_attributes" table
ALTER TABLE "rule_attributes" DROP CONSTRAINT "fk_rule_attributes_attribute", DROP CONSTRAINT "fk_rules_rule_attributes";
-- Modify "rule_conditions" table
ALTER TABLE "rule_conditions" DROP CONSTRAINT "fk_decision_rules_rule_conditions", DROP CONSTRAINT "fk_rule_conditions_attribute";
-- Modify "rules" table
ALTER TABLE "rules" DROP CONSTRAINT "fk_decision_rules_rules";
-- Modify "schedule_occurrences" table
ALTER TABLE "schedule_occurrences" DROP CONSTRAINT "fk_schedule_occurrences_schedule";
-- Modify "schedules" table
ALTER TABLE "schedules" DROP CONSTRAINT "fk_schedules_calendar", DROP CONSTRAINT "fk_schedules_decision_rule", DROP CONSTRAINT "fk_schedules_placement";
-- Modify "users" table
ALTER TABLE "users" DROP CONSTRAINT "fk_users_profile", DROP CONSTRAINT "fk_users_role";
-- Drop "attributes" table
DROP TABLE "attributes";
-- Drop "calendar_dates" table
DROP TABLE "calendar_dates";
-- Drop "calendars" table
DROP TABLE "calendars";
-- Drop "channels" table
DROP TABLE "channels";
-- Drop "clen_schema_registry" table
DROP TABLE "clen_schema_registry";
-- Drop "decision_rule_id_sequences" table
DROP TABLE "decision_rule_id_sequences";
-- Drop "decision_rules" table
DROP TABLE "decision_rules";
-- Drop "login_token_histories" table
DROP TABLE "login_token_histories";
-- Drop "permissions" table
DROP TABLE "permissions";
-- Drop "placements" table
DROP TABLE "placements";
-- Drop "profile_permissions" table
DROP TABLE "profile_permissions";
-- Drop "profiles" table
DROP TABLE "profiles";
-- Drop "roles" table
DROP TABLE "roles";
-- Drop "rule_attributes" table
DROP TABLE "rule_attributes";
-- Drop "rule_conditions" table
DROP TABLE "rule_conditions";
-- Drop "rules" table
DROP TABLE "rules";
-- Drop "schedule_occurrences" table
DROP TABLE "schedule_occurrences";
-- Drop "schedules" table
DROP TABLE "schedules";
-- Drop "users" table
DROP TABLE "users";
