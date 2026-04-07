# Indexes (สำหรับเพิ่ม Performance และ Unique Constraints)
```
-- Unique Indexes ตามข้อกำหนด
CREATE UNIQUE INDEX idx_permissions_feature_action ON permissions (feature_code, action);
CREATE UNIQUE INDEX idx_profile_permissions_unique ON profile_permissions (profile_id, permission_id);
CREATE UNIQUE INDEX idx_schedules_rule_placement ON schedules (decision_rule_id, placement_id);

-- Performance Indexes (แนะนำให้ใส่เพื่อเร่งความเร็วการดึงข้อมูล)
CREATE INDEX idx_users_role_id ON users (role_id);
CREATE INDEX idx_users_profile_id ON users (profile_id);
CREATE INDEX idx_rules_decision_rule_id ON rules (decision_rule_id);
CREATE INDEX idx_rule_conditions_rule_id ON rule_conditions (rule_id);
CREATE INDEX idx_rule_conditions_parent_id ON rule_conditions (parent_rule_condition_id);
CREATE INDEX idx_schedules_active_dates ON schedules (is_active, start_timestamp, end_timestamp);
```