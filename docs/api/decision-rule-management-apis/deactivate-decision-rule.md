# PUT /decision-rules/{id}/deactivate

เปลี่ยนสถานะของ `DecisionRule` จาก `ACTIVE` เป็น `INACTIVE`

---

## Constraints

- ทำได้เฉพาะเมื่อ `status = ACTIVE` เท่านั้น
- หาก `status` เป็น `DRAFT` หรือ `INACTIVE` อยู่แล้ว ระบบจะ return 422
- บันทึก `INACTIVE_BY` = UUID ของ user ที่ทำการ Deactivate (อ่านจาก `X-User-Id` header ผ่าน context)
- `UPDATED_BY` ถูก stamp โดยอัตโนมัติผ่าน GORM `BeforeUpdate` hook

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Request Body

ไม่มี Request Body

---

## Response Body `200 OK`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleRunning": "RS-202604-0001",
    "status": "INACTIVE",
    "updatedAt": "2026-04-27T15:00:00+07:00"
  }
}
```

### Response Field Reference

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | `decision_rules.id` |
| `decisionRuleRunning` | string | Running Number ไม่เปลี่ยนแปลง |
| `status` | string | `INACTIVE` เสมอ |
| `updatedAt` | RFC3339 | เวลาที่ deactivate สำเร็จ |

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| `status = DRAFT` | 422 | `only ACTIVE decision rules can be deactivated, current status is DRAFT` |
| `status = INACTIVE` | 422 | `only ACTIVE decision rules can be deactivated, current status is INACTIVE` |

---

## State Transition

```
DRAFT  ──✗──▶  INACTIVE   (ไม่อนุญาต)
ACTIVE ──✓──▶  INACTIVE   (อนุญาต)
INACTIVE ─✗──▶ INACTIVE   (ไม่อนุญาต — already inactive)
```

---

## DB Mapping

| Column | Table | Value | Notes |
|--------|-------|-------|-------|
| `STATUS` | `decision_rules` | `INACTIVE` | |
| `INACTIVE_BY` | `decision_rules` | `{userId from context}` | UUID ของ user ที่ deactivate |
| `UPDATED_BY` | `decision_rules` | `{userId from context}` | stamp โดย BeforeUpdate hook |
| `UPDATED_AT` | `decision_rules` | `NOW()` | stamp โดย GORM autoUpdateTime |
