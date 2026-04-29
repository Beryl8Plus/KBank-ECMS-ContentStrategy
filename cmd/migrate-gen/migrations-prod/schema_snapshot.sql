table "attributes" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "CLEN_SCHEMA_REGISTRY_ID" {
    null = false
    type = uuid
  }
  column "FIELD_NAME" {
    null = true
    type = character_varying(255)
  }
  column "DISPLAY_NAME" {
    null = true
    type = character_varying(255)
  }
  column "DATA_TYPE" {
    null = true
    type = character_varying(255)
  }
  column "VALUE" {
    null = true
    type = jsonb
  }
  column "DESCRIPTION" {
    null = true
    type = text
  }
  column "SOURCE_SYSTEM" {
    null = true
    type = character_varying(255)
  }
  column "IS_ACTIVE" {
    null    = true
    type    = boolean
    default = true
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_attributes_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "calendar_dates" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "CALENDAR_ID" {
    null = false
    type = uuid
  }
  column "DATE" {
    null = true
    type = date
  }
  column "NAME" {
    null = true
    type = character_varying(255)
  }
  column "IS_RECURRING" {
    null    = true
    type    = boolean
    default = false
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_calendar_dates_calendar" {
    columns     = [column.CALENDAR_ID]
    ref_columns = [table.calendars.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_calendar_dates_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "calendars" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "NAME" {
    null = true
    type = character_varying(255)
  }
  column "TYPE" {
    null = true
    type = character_varying(255)
  }
  column "IS_ACTIVE" {
    null    = true
    type    = boolean
    default = true
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_calendars_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "channels" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "CHANNEL_NAME" {
    null = true
    type = character_varying(255)
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_channels_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "clen_schema_registry" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "SCHEMA_NAME" {
    null = true
    type = character_varying(255)
  }
  column "VERSION" {
    null = true
    type = character_varying(255)
  }
  column "SCHEMA_DEFINITION" {
    null = true
    type = jsonb
  }
  column "IS_ACTIVE" {
    null    = true
    type    = boolean
    default = false
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_clen_schema_registry_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "decision_rule_id_sequences" {
  schema = schema.public
  column "year_month" {
    null = false
    type = text
  }
  column "last_seq" {
    null    = false
    type    = integer
    default = 0
  }
  primary_key "pk_decision_rule_id_seq" {
    columns = [column.year_month]
  }
}
table "decision_rules" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "DECISION_RULE_RUNNING" {
    null = false
    type = character_varying(255)
  }
  column "NAME" {
    null = false
    type = character_varying(255)
  }
  column "TYPE" {
    null = false
    type = character_varying(255)
  }
  column "EVALUATE_TYPE" {
    null = false
    type = character_varying(255)
  }
  column "CONTENT_PATH" {
    null = false
    type = character_varying(255)
  }
  column "CAMPAIGN_CODE" {
    null = true
    type = character_varying(25)
  }
  column "SCORE" {
    null    = true
    type    = numeric
    default = 0
  }
  column "STATUS" {
    null = true
    type = character_varying(255)
  }
  column "SUB_STATUS" {
    null = true
    type = character_varying(255)
  }
  column "INACTIVE_BY" {
    null = true
    type = uuid
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_decision_rules_inactive_by_user" {
    columns     = [column.INACTIVE_BY]
    ref_columns = [table.users.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_decision_rules_active_status" {
    columns = [column.STATUS]
    where   = "(((\"STATUS\")::text = 'ACTIVE'::text) AND (\"DELETED_AT\" IS NULL))"
  }
  index "idx_decision_rules_decision_rule_running" {
    unique  = true
    columns = [column.DECISION_RULE_RUNNING]
  }
  index "idx_decision_rules_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "login_token_histories" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "USER_NAME" {
    null = true
    type = character_varying(255)
  }
  column "ACCESS_TOKEN" {
    null = true
    type = character_varying(255)
  }
  column "EXPIRE_DATE" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_login_token_histories_deleted_at" {
    columns = [column.DELETED_AT]
  }
  index "idx_login_token_histories_user_name" {
    unique  = true
    columns = [column.USER_NAME]
  }
}
table "permissions" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "NAME" {
    null = true
    type = character_varying(255)
  }
  column "PERMISSION_TYPE" {
    null = true
    type = character_varying(255)
  }
  column "SOURCE" {
    null = true
    type = character_varying(255)
  }
  column "ACTION" {
    null = true
    type = character_varying(255)
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_permissions_deleted_at" {
    columns = [column.DELETED_AT]
  }
  index "idx_source_action" {
    unique  = true
    columns = [column.SOURCE, column.ACTION]
  }
}
table "placements" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "PLACEMENT_NAME" {
    null = false
    type = character_varying(255)
  }
  column "CHANNEL_ID" {
    null = false
    type = uuid
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_placements_channel" {
    columns     = [column.CHANNEL_ID]
    ref_columns = [table.channels.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_channel_placement_name" {
    unique  = true
    columns = [column.PLACEMENT_NAME, column.CHANNEL_ID]
  }
  index "idx_placements_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "profile_permissions" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "PROFILE_ID" {
    null = true
    type = uuid
  }
  column "PERMISSION_ID" {
    null = true
    type = uuid
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_profile_permissions_permission" {
    columns     = [column.PERMISSION_ID]
    ref_columns = [table.permissions.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "fk_profile_permissions_profile" {
    columns     = [column.PROFILE_ID]
    ref_columns = [table.profiles.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_profile_permission" {
    unique  = true
    columns = [column.PROFILE_ID, column.PERMISSION_ID]
  }
  index "idx_profile_permissions_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "profiles" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "NAME" {
    null = true
    type = character_varying(255)
  }
  column "CODE" {
    null = true
    type = character_varying(255)
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_profiles_code" {
    unique  = true
    columns = [column.CODE]
  }
  index "idx_profiles_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "roles" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "NAME" {
    null = true
    type = character_varying(255)
  }
  column "CODE" {
    null = true
    type = character_varying(255)
  }
  primary_key {
    columns = [column.ID]
  }
  index "idx_roles_code" {
    unique  = true
    columns = [column.CODE]
  }
  index "idx_roles_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "rule_attributes" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "RULE_ID" {
    null = false
    type = uuid
  }
  column "ATTRIBUTE_ID" {
    null = false
    type = uuid
  }
  column "VALUE" {
    null = true
    type = jsonb
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_rule_attributes_attribute" {
    columns     = [column.ATTRIBUTE_ID]
    ref_columns = [table.attributes.column.ID]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "fk_rules_rule_attributes" {
    columns     = [column.RULE_ID]
    ref_columns = [table.rules.column.ID]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  index "idx_rule_attributes_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "rule_conditions" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "SEQUENCE" {
    null = true
    type = integer
  }
  column "DECISION_RULE_ID" {
    null = false
    type = uuid
  }
  column "PARENT_RULE_CONDITION_ID" {
    null = true
    type = uuid
  }
  column "ATTRIBUTE_ID" {
    null = false
    type = uuid
  }
  column "LOGICAL_OPERATOR" {
    null = true
    type = character_varying(50)
  }
  column "CONNECTOR_OPERATOR" {
    null = true
    type = character_varying(50)
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_decision_rules_rule_conditions" {
    columns     = [column.DECISION_RULE_ID]
    ref_columns = [table.decision_rules.column.ID]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "fk_rule_conditions_attribute" {
    columns     = [column.ATTRIBUTE_ID]
    ref_columns = [table.attributes.column.ID]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "fk_rule_conditions_rule_condition_children" {
    columns     = [column.PARENT_RULE_CONDITION_ID]
    ref_columns = [table.rule_conditions.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_rule_conditions_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "rules" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "DECISION_RULE_ID" {
    null = false
    type = uuid
  }
  column "VARIATION_NAME" {
    null = true
    type = character_varying(255)
  }
  column "SCORE" {
    null = true
    type = numeric
  }
  column "ORDER_NO" {
    null = true
    type = integer
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_decision_rules_rules" {
    columns     = [column.DECISION_RULE_ID]
    ref_columns = [table.decision_rules.column.ID]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  index "idx_rules_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "schedule_occurrences" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "SCHEDULE_ID" {
    null = false
    type = uuid
  }
  column "OCCURRENCE_START" {
    null = true
    type = timestamptz
  }
  column "OCCURRENCE_END" {
    null = true
    type = timestamptz
  }
  column "STATUS" {
    null = true
    type = character_varying(255)
  }
  column "SOURCE" {
    null = true
    type = character_varying(255)
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_schedule_occurrences_schedule" {
    columns     = [column.SCHEDULE_ID]
    ref_columns = [table.schedules.column.ID]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  index "idx_occurrence_schedule_start_end" {
    unique  = true
    columns = [column.SCHEDULE_ID, column.OCCURRENCE_START, column.OCCURRENCE_END]
  }
  index "idx_schedule_occurrences_deleted_at" {
    columns = [column.DELETED_AT]
  }
}
table "schedules" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "DECISION_RULE_ID" {
    null = true
    type = uuid
  }
  column "PLACEMENT_ID" {
    null = true
    type = uuid
  }
  column "CALENDAR_ID" {
    null = true
    type = uuid
  }
  column "RECURRENCE_TYPE" {
    null = true
    type = character_varying(255)
  }
  column "RECURRENCE_RULE" {
    null = true
    type = text
  }
  column "EFFECTIVE_FROM" {
    null = true
    type = timestamptz
  }
  column "EFFECTIVE_UNTIL" {
    null = true
    type = timestamptz
  }
  column "TIME_OF_DAY_START" {
    null = true
    type = character_varying(5)
  }
  column "TIME_OF_DAY_END" {
    null = true
    type = character_varying(5)
  }
  column "ALL_DAY" {
    null    = true
    type    = boolean
    default = false
  }
  column "TIMEZONE" {
    null    = true
    type    = character_varying(255)
    default = "Asia/Bangkok"
  }
  column "IS_ACTIVE" {
    null    = true
    type    = boolean
    default = false
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_schedules_calendar" {
    columns     = [column.CALENDAR_ID]
    ref_columns = [table.calendars.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "fk_schedules_decision_rule" {
    columns     = [column.DECISION_RULE_ID]
    ref_columns = [table.decision_rules.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "fk_schedules_placement" {
    columns     = [column.PLACEMENT_ID]
    ref_columns = [table.placements.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_schedules_active_window" {
    columns = [column.EFFECTIVE_FROM, column.EFFECTIVE_UNTIL]
    where   = "((\"IS_ACTIVE\" = true) AND (\"DELETED_AT\" IS NULL))"
  }
  index "idx_schedules_created_at_desc_active" {
    where = "(\"DELETED_AT\" IS NULL)"
    on {
      desc   = true
      column = column.CREATED_AT
    }
  }
  index "idx_schedules_deleted_at" {
    columns = [column.DELETED_AT]
  }
  exclude "no_overlap_active_schedule_per_rule_placement" {
    type  = GIST
    where = "((\"IS_ACTIVE\" = true) AND (\"DELETED_AT\" IS NULL))"
    on {
      column = column.DECISION_RULE_ID
      op     = "="
    }
    on {
      column = column.PLACEMENT_ID
      op     = "="
    }
    on {
      expr = "tstzrange(\"EFFECTIVE_FROM\", \"EFFECTIVE_UNTIL\")"
      op   = "&&"
    }
  }
}
table "users" {
  schema = schema.public
  column "ID" {
    null    = false
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "CREATED_AT" {
    null = true
    type = timestamptz
  }
  column "CREATED_BY" {
    null = true
    type = uuid
  }
  column "UPDATED_AT" {
    null = true
    type = timestamptz
  }
  column "UPDATED_BY" {
    null = true
    type = uuid
  }
  column "DELETED_AT" {
    null = true
    type = timestamptz
  }
  column "ROLE_ID" {
    null = true
    type = uuid
  }
  column "PROFILE_ID" {
    null = true
    type = uuid
  }
  column "EMAIL" {
    null = true
    type = character_varying(255)
  }
  column "NAME_TH" {
    null = true
    type = character_varying(255)
  }
  column "NAME_EN" {
    null = true
    type = character_varying(255)
  }
  column "IS_ACTIVE" {
    null    = true
    type    = boolean
    default = true
  }
  primary_key {
    columns = [column.ID]
  }
  foreign_key "fk_users_profile" {
    columns     = [column.PROFILE_ID]
    ref_columns = [table.profiles.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "fk_users_role" {
    columns     = [column.ROLE_ID]
    ref_columns = [table.roles.column.ID]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_users_deleted_at" {
    columns = [column.DELETED_AT]
  }
  index "idx_users_email" {
    unique  = true
    columns = [column.EMAIL]
  }
}
schema "public" {
  comment = "standard public schema"
}
