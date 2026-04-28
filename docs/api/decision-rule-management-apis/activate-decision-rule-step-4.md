# PUT /decision-rules/{id}/activate

เปิดใช้งาน `DecisionRule` (Wizard Step 4)

---

## Constraints

- `DecisionRule` ต้องมี Schedule อย่างน้อย 1 รายการ (ผ่าน Step 3 แล้ว)
- **Integrity Check ก่อน Activate:** ระบบตรวจสอบ 2 เงื่อนไขต่อไปนี้ก่อน set `status = ACTIVE`:
  1. **Active Attributes:** ทุก `attributeId` ใน `rule_conditions` ของ Rule นี้ต้องมี `is_active = true`
  2. **Valid Values:** ทุก `value` ใน `rule_attributes` ต้องยังคงอยู่ใน allowed options JSON ของ Attribute นั้น
- ถ้าผ่านทั้ง 2 เงื่อนไข: `status → ACTIVE`, **`subStatus → "N/A"`** (reset ทุกครั้ง)
- ถ้าไม่ผ่าน: return 422 พร้อม error message; `status` ไม่เปลี่ยน

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Request Body

ไม่มี request body

---

## Response Body `200 OK`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleId": "DR-20260424-001",
    "status": "ACTIVE",
    "schedules": [
      {
        "id": "s1000000-0000-0000-0000-000000000001",
        "placementId": "p1000000-0000-0000-0000-000000000001",
        "effectiveFrom": "2026-04-30T17:00:00Z",
        "effectiveUntil": "2026-05-31T16:59:59Z",
        "recurrenceType": "ONCE",
        "allDay": true,
        "isActive": true
      }
    ]
  }
}
```

### Response Field Reference

| Field | Source | Notes |
|-------|--------|-------|
| `id` | `decision_rules.id` | UUID primary key |
| `decisionRuleId` | `decision_rules.decision_rule_id` | business key |
| `status` | `decision_rules.status` | always `ACTIVE` on success |
| `schedules` | `schedules` WHERE `decision_rule_id = {id}` | รายการ schedule ทั้งหมดของ rule นี้ |

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| ยังไม่มี Schedule (Step 3 ยังไม่เสร็จ) | 422 | `decision rule has no schedules, complete step 3 first` |
| มี Attribute ที่ถูก deactivate อยู่ใน Rule | 422 | `attribute {id} ({fieldName}) is inactive — update the rule before activating` |
| มี RuleAttribute value ที่ไม่อยู่ใน allowed options | 422 | `rule value "{value}" is no longer in the allowed options for attribute {id}` |

---

## DB Operations

```sql
-- Integrity Check 1: inactive attributes in rule_conditions
SELECT DISTINCT rc.DECISION_RULE_ID
FROM rule_conditions rc
JOIN attributes a ON a.ID = rc.ATTRIBUTE_ID AND a.DELETED_AT IS NULL
WHERE rc.DELETED_AT IS NULL
  AND rc.DECISION_RULE_ID = :id
  AND a.IS_ACTIVE = false;
-- → reject if any rows returned

-- Integrity Check 2: invalid values in rule_attributes
SELECT ra.ID
FROM rule_attributes ra
JOIN attributes a  ON a.ID  = ra.ATTRIBUTE_ID AND a.DELETED_AT IS NULL
JOIN rules r       ON r.ID  = ra.RULE_ID       AND r.DELETED_AT IS NULL
WHERE ra.DELETED_AT IS NULL
  AND r.DECISION_RULE_ID = :id
  AND a.VALUE IS NOT NULL
  AND jsonb_typeof(a.VALUE) = 'array'
  AND jsonb_array_length(a.VALUE) > 0
  AND NOT (a.VALUE @> ra.VALUE);
-- → reject if any rows returned

-- Activate + reset SubStatus
UPDATE decision_rules
SET    STATUS     = 'ACTIVE',
       SUB_STATUS = 'N/A',
       UPDATED_AT = NOW()
WHERE  ID = :id
  AND  DELETED_AT IS NULL;
```

---

## SubStatus Reset Behaviour

| สถานการณ์ | Status ก่อน | SubStatus ก่อน | Status หลัง | SubStatus หลัง |
|-----------|-------------|----------------|-------------|----------------|
| Activate ปกติ | DRAFT | N/A | ACTIVE | **N/A** |
| Re-activate หลังแก้ไข (เคย INACTIVE / Missing) | INACTIVE | Missing attribute registry | ACTIVE | **N/A** ← reset |
| Integrity check fail | ใดก็ได้ | ใดก็ได้ | ไม่เปลี่ยน | ไม่เปลี่ยน |

> `subStatus` จะถูก reset เป็น `"N/A"` **ทุกครั้ง** ที่ activate สำเร็จ
> ไม่ว่าก่อนหน้านี้จะเป็น `"Missing attribute registry"` หรือไม่ก็ตาม

---

## Background Integrity Checker

`AttributeSyncWorker` รัน 2 งานอัตโนมัติในพื้นหลัง:

| งาน | รอบ | หน้าที่ |
|-----|-----|--------|
| **Attribute Sync** | ทุกวัน **ตี 3** (03:00 local time) | ดึง schema ล่าสุดจาก External API → Bulk upsert `attributes` → Deactivate attribute ที่หายไป |
| **Integrity Check** | ทุก **5 นาที** | ตรวจสอบ Rule ที่ใช้ inactive attribute หรือ value ที่ไม่อยู่ใน allowed set |

เมื่อ Integrity Check พบ Rule ที่ผิดพลาด จะ:

1. Set `status = INACTIVE`, `sub_status = "Missing attribute registry"` อัตโนมัติ
2. User แก้ไข Rule ให้ถูกต้อง (Step 1/2)
3. User เรียก Step 4 อีกครั้ง → Integrity Check ผ่าน → `status = ACTIVE`, `sub_status = "N/A"`

> Integrity Check ยังรันทันทีเมื่อ service เริ่มต้น (startup) เพื่อ detect ปัญหาที่อาจเกิดขึ้นระหว่าง downtime
