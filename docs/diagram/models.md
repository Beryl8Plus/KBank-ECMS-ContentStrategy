documentation for database models used in the system. This includes tables for management & authentication, decision rule core, advanced logic engine, attributes & sources, and delivery & scheduling.

# Database Models

The system's database schema is organized into several key areas, each serving a specific purpose in the overall architecture. Below is an overview of the main tables and their relationships.
[https://dbdiagram.io/d/](https://dbdiagram.io/d/)

```
// --- 1. Management & Auth ---
Table login_token_histories {
    id uuid [primary key]
    username varchar [unique]
    access_token varchar
    expire_date timestamptz

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
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
    deleted_at timestamptz
}

Table roles {
    id uuid [primary key]
    name varchar
    code varchar [unique]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

Table profiles {
    id uuid [primary key]
    name varchar
    code varchar [unique]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

Table permissions {
    id uuid [primary key]
    name varchar
    permission_type enum [note: 'ACCESS_CONTROL, FEATURE_FLAG, etc.']
    feature_code varchar [note: 'e.g., Content_Decision_Rule']
    action varchar [note: 'CREATE, EDIT, DELETE, VIEW ALL, EDIT ALL, DELETE ALL']

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz

    indexes {
        (feature_code, action) [unique]
    }
}

Table profile_permissions {
    id uuid [primary key]
    profile_id uuid [ref: > profiles.id]
    permission_id uuid [ref: > permissions.id]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz

    indexes {
        (profile_id, permission_id) [unique]
    }
}

// --- 2. Decision Rule Core ---
Table decision_rules {
    id uuid [primary key]
    name varchar
    type enum [note: 'SCORING, SEGMENT, ELIGIBLE']
    content_path varchar
    score float
    status enum [note: 'DRAFT, ACTIVE, INACTIVE']

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz

    inactive_by uuid [ref: > users.id]
}

Table rules {
    id uuid [primary key]
    decision_rule_id uuid [ref: > decision_rules.id]
    variation_name varchar
    score integer
    order_no integer // asc น้อยไปมาก

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

// --- 3. Advanced Logic Engine ---
Table rule_conditions {
    id uuid [primary key]
    sequence integer
    decision_rule_id uuid [ref: > decision_rules.id]
    rule_id uuid [ref: > rules.id]
    parent_rule_condition_id uuid [ref: > rule_conditions.id]
    attribute_id uuid [ref: > attributes.id]
    logical_operator enum // <,>,=,IN,BETWEEN
    value jsonb
    connector_operator enum [note: 'AND, OR — connects this condition to the next item in sequence. null for last item in group.']

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

// --- 5. Attributes & Sources ---
Table attributes {
    id uuid [primary key]
    field_name varchar
    display_name varchar // display name of attribute
    data_type enum // Text, Date, Number, Boolean
    value varchar // Possible value
    description text // optional
    source_system varchar
    is_active boolean [default: true]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

// --- 6. Delivery & Scheduling ---
Table placements { // masters data
    id uuid [primary key]
    name varchar
    description text

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

Table schedules {
    id uuid [primary key]
    decision_rule_id uuid [ref: > decision_rules.id]
    placement_id uuid [ref: > placements.id]
    calendar_id uuid [ref: > calendars.id]
    recurrence_type enum [note: 'ONCE, RRULE, CALENDAR']
    recurrence_rule text
    effective_from timestamptz
    effective_until timestamptz
    time_of_day_start varchar [note: 'HH:mm']
    time_of_day_end varchar [note: 'HH:mm']
    all_day boolean [default: false]
    timezone varchar [default: 'Asia/Bangkok']
    is_active boolean [default: false]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz

    indexes {
        (decision_rule_id, placement_id) [unique]
    }
}

Table schedule_occurrences {
    id uuid [primary key]
    schedule_id uuid [ref: > schedules.id]
    occurrence_start timestamptz
    occurrence_end timestamptz
    status enum [note: 'ACTIVE, CANCELLED, MODIFIED']
    source enum [note: 'RECURRENCE, CALENDAR, MANUAL']

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

Table calendars {
    id uuid [primary key]
    name varchar
    type enum [note: 'HOLIDAY, PERSONAL, CUSTOM']
    is_active boolean [default: true]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

Table calendar_dates {
    id uuid [primary key]
    calendar_id uuid [ref: > calendars.id]
    date date
    name varchar
    is_recurring boolean [default: false]

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}

// possible value for display frontend
Table mdp_schema_registry {
    id uuid [primary key]
    schema_name varchar
    version varchar
    schema_definition jsonb
    is_active boolean

    created_at timestamptz
    created_by uuid [ref: > users.id]
    updated_at timestamptz
    updated_by uuid [ref: > users.id]
    deleted_at timestamptz
}
```
