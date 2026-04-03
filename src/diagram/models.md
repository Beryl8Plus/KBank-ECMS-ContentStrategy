// --- 1. Management & Auth ---
Table login_token_histories {
id uuid [primary key]
username varchar [unique]
access_token varchar
expire_date timestamptz
}

Table users {
id uuid [primary key]
role_id uuid [ref: > roles.id]
profile_id uuid [ref: - profiles.id]
email varchar [unique]
name_th varchar
name_en varchar
is_active boolean [default: true]
created_at timestamptz
created_by uuid [ref: > users.id]
updated_at timestamptz
updated_by uuid [ref: > users.id]
}

Table roles {
id uuid [primary key]
name varchar
code varchar [unique]
created_at timestamptz
created_by uuid [ref: > users.id]
updated_at timestamptz
updated_by uuid [ref: > users.id]
}

Table profiles {
id uuid [primary key]
name varchar
code varchar [unique]
created_at timestamptz
created_by uuid [ref: > users.id]
updated_at timestamptz
updated_by uuid [ref: > users.id]
}

Table permissions {
id uuid [primary key]
name varchar [note: 'Sample Name']
source varchar [note: 'เช่น Content_Decision_Rule']
action varchar [note: 'CREATE, EDIT, DELETE, VIEW ALL, EDIT ALL, DELETE ALL']
created_at timestamptz
created_by uuid [ref: > users.id]
updated_at timestamptz
updated_by uuid [ref: > users.id]

indexes {
(source, action) [unique]
}
}

Table profile_permisssions {
id uuid [primary key]
profile_id uuid [ref: > profiles.id]
permission_id uuid [ref: > permissions.id]
indexes {
(profile_id, permission_id) [unique]
}
}

// --- 2. Decision Rule Core ---
Table decision_rules {
id uuid [primary key]
name varchar
// description varchar
type enum
content_path varchar
score integer
status enum // DRAFT, ACTIVE, INACTIVE
created_at timestamptz
created_by uuid [ref: > users.id]
updated_at timestamptz
updated_by uuid [ref: > users.id]

inactive_by varchar // waiting confirm
}

Table rules {
id uuid [primary key]
decision_rule_id uuid [ref: > decision_rules.id]
variation_name varchar
score integer
order_no integer // asc น้อยไปมาก
}

// --- 3. Advanced Logic Engine ---
Table rule_condition_groups {
id uuid [primary key]
parent_rule_condition_groups uuid [ref: > rule_condition_groups.id]
// rule_operator removed — group is visual container only, operators live on rule_condition.connector_operator
}

Table rule_condition {
id uuid [primary key]
sequence integer
decision_rule_id uuid [ref: > decision_rules.id]
rule_id uuid [ref: > rules.id]
rule_condition_group_id uuid [ref: > rule_condition_groups.id]
attribute_id uuid [ref: > attributes.id]
logical_operator enum // <,>,=,IN,BETWEEN
value jsonb
connector_operator enum [note: 'AND, OR — connects this condition to the next item in sequence. null for last item in group.']
}

// --- 5. Attributes & Sources ---
Table attributes {
id uuid [primary key]
created_at timestamptz
created_by uuid [ref: > users.id]
updated_at timestamptz
updated_by uuid [ref: > users.id]
is_active boolean [default: true]

field_name varchar
display_name varchar // display name of attribute
data_type enum // Text, Date, Number, Boolean
value varchar // Possible value
description text // optional
source_system varchar

}

// --- 6. Delivery & Scheduling ---
Table placements { // masters data
id uuid [primary key]
name varchar
description text
}

Table schedules {
id uuid [primary key]
decision_rule_id uuid [ref: > decision_rules.id]
placement_id uuid [ref: > placements.id]
start_timestamp timestamptz
end_timestamp timestamptz
is_active boolean

indexes {
(decision_rule_id, placement_id) [unique]
}
}

// possible value for display frontend
Table schema_registry {
id uuid [primary key]
schema_name varchar
version varchar
schema_definition jsonb
is_active boolean
created_at timestamp
}
