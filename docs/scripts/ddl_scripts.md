# DDL Scripts (Data Definition Language)
```sql
-- ==========================================
-- 0. CREATE ENUM TYPES
-- ==========================================
CREATE TYPE permission_type AS ENUM ('ACCESS_CONTROL', 'FEATURE_FLAG');
CREATE TYPE decision_rule_type AS ENUM ('SCORING', 'SEGMENT', 'ELIGIBLE');
CREATE TYPE decision_rule_status AS ENUM ('DRAFT', 'ACTIVE', 'INACTIVE');
CREATE TYPE logical_operator AS ENUM ('<', '>', '=', '!=', '<=', '>=', 'IN', 'BETWEEN');
CREATE TYPE connector_operator AS ENUM ('AND', 'OR');
CREATE TYPE attribute_data_type AS ENUM ('TEXT', 'DATE', 'NUMBER', 'BOOLEAN');
CREATE TYPE recurrence_type AS ENUM ('NONE', 'DAILY', 'WEEKLY', 'MONTHLY', 'YEARLY');
CREATE TYPE schedule_status AS ENUM ('SCHEDULED', 'COMPLETED', 'CANCELLED');
CREATE TYPE schedule_source AS ENUM ('GENERATED', 'MANUAL');
CREATE TYPE calendar_type AS ENUM ('HOLIDAY', 'SPECIAL_DATE');

-- ==========================================
-- 1. MANAGEMENT & AUTHENTICATION
-- ==========================================
CREATE TABLE login_token_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE,
    access_token VARCHAR(1000),
    expire_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID,
    profile_id UUID,
    email VARCHAR(255) UNIQUE NOT NULL,
    name_th VARCHAR(255),
    name_en VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    code VARCHAR(100) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    code VARCHAR(100) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    permission_type permission_type,
    feature_code VARCHAR(255),
    action VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE profile_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID,
    permission_id UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 2. MDP SCHEMA REGISTRY (สร้างก่อนเพราะ Attributes อ้างอิง)
-- ==========================================
CREATE TABLE mdp_schema_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_name VARCHAR(255),
    version VARCHAR(50),
    schema_definition JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 3. ATTRIBUTES & SOURCES
-- ==========================================
CREATE TABLE attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_name VARCHAR(255),
    display_name VARCHAR(255),
    data_type attribute_data_type,
    value JSONB,
    description TEXT,
    source_system VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    mdp_schema_registry_id UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 4. DECISION RULE CORE & ADVANCED LOGIC
-- ==========================================
CREATE TABLE decision_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    type decision_rule_type,
    content_path VARCHAR(500),
    score DECIMAL(11,2),
    status decision_rule_status DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    inactive_by UUID
);

CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_rule_id UUID,
    variation_name VARCHAR(255),
    score INTEGER,
    order_no INTEGER,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE rule_conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sequence INTEGER,
    decision_rule_id UUID,
    rule_id UUID,
    parent_rule_condition_id UUID,
    attribute_id UUID,
    logical_operator logical_operator,
    value JSONB,
    connector_operator connector_operator,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 5. DELIVERY & SCHEDULING
-- ==========================================
CREATE TABLE placements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    description TEXT,
    source VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    type calendar_type,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE calendar_dates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID,
    date DATE,
    name VARCHAR(255),
    is_recurring BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_rule_id UUID,
    placement_id UUID,
    calendar_id UUID,
    recurrence_type recurrence_type,
    recurrence_rule TEXT,
    effective_from TIMESTAMPTZ,
    effective_until TIMESTAMPTZ,
    time_of_day_start VARCHAR(5),
    time_of_day_end VARCHAR(5),
    all_day BOOLEAN DEFAULT false,
    timezone VARCHAR(50) DEFAULT 'Asia/Bangkok',
    is_active BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE schedule_occurrences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID,
    occurrence_start TIMESTAMPTZ,
    occurrence_end TIMESTAMPTZ,
    status schedule_status,
    source schedule_source,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_at TIMESTAMPTZ,
    updated_by UUID,
    deleted_at TIMESTAMPTZ
);

-- Management & Auth
ALTER TABLE users ADD CONSTRAINT fk_users_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL;
ALTER TABLE users ADD CONSTRAINT fk_users_profile FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE SET NULL;
ALTER TABLE profile_permissions ADD CONSTRAINT fk_pp_profile FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE;
ALTER TABLE profile_permissions ADD CONSTRAINT fk_pp_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;

-- Attributes & MDP
ALTER TABLE attributes ADD CONSTRAINT fk_attributes_mdp FOREIGN KEY (mdp_schema_registry_id) REFERENCES mdp_schema_registry(id);

-- Decision Rule Core & Logic
ALTER TABLE rules ADD CONSTRAINT fk_rules_decision_rule FOREIGN KEY (decision_rule_id) REFERENCES decision_rules(id) ON DELETE CASCADE;
ALTER TABLE rule_conditions ADD CONSTRAINT fk_rc_decision_rule FOREIGN KEY (decision_rule_id) REFERENCES decision_rules(id) ON DELETE CASCADE;
ALTER TABLE rule_conditions ADD CONSTRAINT fk_rc_rule FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE;
ALTER TABLE rule_conditions ADD CONSTRAINT fk_rc_parent FOREIGN KEY (parent_rule_condition_id) REFERENCES rule_conditions(id) ON DELETE CASCADE;
ALTER TABLE rule_conditions ADD CONSTRAINT fk_rc_attribute FOREIGN KEY (attribute_id) REFERENCES attributes(id);

-- Delivery & Scheduling
ALTER TABLE calendar_dates ADD CONSTRAINT fk_cd_calendar FOREIGN KEY (calendar_id) REFERENCES calendars(id) ON DELETE CASCADE;
ALTER TABLE schedules ADD CONSTRAINT fk_schedules_decision_rule FOREIGN KEY (decision_rule_id) REFERENCES decision_rules(id) ON DELETE CASCADE;
ALTER TABLE schedules ADD CONSTRAINT fk_schedules_placement FOREIGN KEY (placement_id) REFERENCES placements(id) ON DELETE CASCADE;
ALTER TABLE schedules ADD CONSTRAINT fk_schedules_calendar FOREIGN KEY (calendar_id) REFERENCES calendars(id) ON DELETE SET NULL;
ALTER TABLE schedule_occurrences ADD CONSTRAINT fk_so_schedule FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE;
```