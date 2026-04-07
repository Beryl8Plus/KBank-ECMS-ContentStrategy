# Example Queries (ตัวอย่างการ Query ไปใช้งานจริง)

## 1. ดึงข้อมูลผู้ใช้งาน พร้อมระบุ Profile และสิทธิ์ทั้งหมดที่มี (Auth Check)
```
SELECT 
    u.email, 
    u.name_en, 
    pr.name AS profile_name,
    p.feature_code, 
    p.action
FROM users u
JOIN profiles pr ON u.profile_id = pr.id
JOIN profile_permissions pp ON pr.id = pp.profile_id
JOIN permissions p ON pp.permission_id = p.id
WHERE u.email = 'admin@example.com' 
  AND u.is_active = true 
  AND u.deleted_at IS NULL;
```
## 2. ดึงโครงสร้างของ Decision Rule พร้อม Condition เรียงตามลำดับ (Rule Engine Flow)
```
SELECT 
    dr.name AS rule_group_name,
    r.variation_name,
    rc.sequence,
    attr.field_name,
    rc.logical_operator,
    rc.value AS compare_value,
    rc.connector_operator
FROM decision_rules dr
JOIN rules r ON dr.id = r.decision_rule_id
JOIN rule_conditions rc ON r.id = rc.rule_id
JOIN attributes attr ON rc.attribute_id = attr.id
WHERE dr.id = 'ใส่-UUID-ของ-Decision-Rule'
  AND dr.status = 'ACTIVE'
ORDER BY r.order_no ASC, rc.sequence ASC;
```
## 3. ตรวจสอบว่า Decision Rule ไหนบ้างที่กำลังถูก Schedule ให้แสดงผลอยู่ ณ ตอนนี้
```
SELECT 
    dr.name AS rule_name, 
    pl.name AS placement_name, 
    s.start_timestamp, 
    s.end_timestamp
FROM schedules s
JOIN decision_rules dr ON s.decision_rule_id = dr.id
JOIN placements pl ON s.placement_id = pl.id
WHERE s.is_active = true
  AND CURRENT_TIMESTAMP BETWEEN s.start_timestamp AND s.end_timestamp;
```