# DDL Scripts (Data Definition Language)
```
-- ==========================================
-- 0. CREATE ENUM TYPES
-- ==========================================
CREATE TYPE permission_type AS ENUM ('ACCESS_CONTROL', 'FEATURE_FLAG', 'OTHER');
CREATE TYPE decision_rule_type AS ENUM ('SCORING', 'SEGMENT', 'ELIGIBLE');
CREATE TYPE decision_rule_status AS ENUM ('DRAFT', 'ACTIVE', 'INACTIVE');
CREATE TYPE logical_operator AS ENUM ('<', '>', '=', '!=', '<=', '>=', 'IN', 'BETWEEN');
CREATE TYPE connector_operator AS ENUM ('AND', 'OR');
CREATE TYPE attribute_data_type AS ENUM ('TEXT', 'DATE', 'NUMBER', 'BOOLEAN');

-- ==========================================
-- 1. MANAGEMENT & AUTHENTICATION
-- ==========================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    code VARCHAR(100) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    code VARCHAR(100) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    permission_type permission_type,
    feature_code VARCHAR(255),
    action VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE profile_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID REFERENCES profiles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    profile_id UUID REFERENCES profiles(id) ON DELETE SET NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    name_th VARCHAR(255),
    name_en VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE login_token_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255),
    access_token VARCHAR(1000),
    expire_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 2. ATTRIBUTES & SOURCES (สร้างก่อนเพราะ Rule ต้องอ้างอิง)
-- ==========================================
CREATE TABLE attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_name VARCHAR(255),
    display_name VARCHAR(255),
    data_type attribute_data_type,
    value VARCHAR(255),
    description TEXT,
    source_system VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 3. DECISION RULE CORE & ADVANCED LOGIC
-- ==========================================
CREATE TABLE decision_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    type decision_rule_type,
    content_path VARCHAR(500),
    score FLOAT,
    status decision_rule_status DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ,
    inactive_by UUID REFERENCES users(id)
);

CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_rule_id UUID REFERENCES decision_rules(id) ON DELETE CASCADE,
    variation_name VARCHAR(255),
    score INTEGER,
    order_no INTEGER,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE rule_conditions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sequence INTEGER,
    decision_rule_id UUID REFERENCES decision_rules(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES rules(id) ON DELETE CASCADE,
    parent_rule_condition_id UUID REFERENCES rule_conditions(id) ON DELETE CASCADE,
    attribute_id UUID REFERENCES attributes(id),
    logical_operator logical_operator,
    value JSONB,
    connector_operator connector_operator,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 4. DELIVERY & SCHEDULING
-- ==========================================
CREATE TABLE placements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_rule_id UUID REFERENCES decision_rules(id) ON DELETE CASCADE,
    placement_id UUID REFERENCES placements(id) ON DELETE CASCADE,
    start_timestamp TIMESTAMPTZ,
    end_timestamp TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);

-- ==========================================
-- 5. MDP SCHEMA REGISTRY
-- ==========================================
CREATE TABLE mdp_schema_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_name VARCHAR(255),
    version VARCHAR(50),
    schema_definition JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id),
    deleted_at TIMESTAMPTZ
);
```