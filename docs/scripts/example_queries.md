# Example Queries (ตัวอย่างการ Query ไปใช้งานจริง)

## 1. ดึงกฎทั้งหมดของหน้า Placement "Home Page Banner" ที่มี Schedule อนุมัติและกำลัง Active อยู่ตอนนี้
```sql
SELECT 
    dr.id AS rule_id,
    dr.name AS rule_name,
    dr.type AS rule_type,
    dr.content_path,
    s.time_of_day_start,
    s.time_of_day_end
FROM schedules s
JOIN decision_rules dr ON s.decision_rule_id = dr.id
JOIN placements p ON s.placement_id = p.id
WHERE p.name = 'Home Page Banner'
  AND s.is_active = true
  AND dr.status = 'ACTIVE'
  AND CURRENT_TIMESTAMP BETWEEN s.effective_from AND s.effective_until;
```
## 2. ตรวจสอบเงื่อนไข (Conditions) ของ Rule ที่สนใจ พร้อมเรียงลำดับการทำงาน (Sequence)
```sql
SELECT 
    r.variation_name,
    rc.sequence,
    a.field_name AS attribute_name,
    rc.logical_operator,
    rc.value AS compare_value,
    rc.connector_operator
FROM rules r
JOIN rule_conditions rc ON r.id = rc.rule_id
JOIN attributes a ON rc.attribute_id = a.id
WHERE r.decision_rule_id = 'ใส่-UUID-ของ-Decision-Rule'
ORDER BY r.order_no ASC, rc.sequence ASC;
```
## 3. ดึงรอบเวลา (Occurrences) ที่กำลังจะเกิดขึ้นในสัปดาห์หน้า (ตรวจสอบว่า Cronjob แตก Schedule ไว้ถูกต้องหรือไม่)
```sql
SELECT 
    dr.name AS rule_name,
    so.occurrence_start,
    so.occurrence_end,
    so.status,
    so.source
FROM schedule_occurrences so
JOIN schedules s ON so.schedule_id = s.id
JOIN decision_rules dr ON s.decision_rule_id = dr.id
WHERE so.occurrence_start >= CURRENT_TIMESTAMP
  AND so.occurrence_start <= CURRENT_TIMESTAMP + INTERVAL '7 days'
  AND so.status = 'SCHEDULED'
ORDER BY so.occurrence_start ASC;
```