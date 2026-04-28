# DELETE /decision-rules/{id}

Soft-delete `DecisionRule` และข้อมูลลูกทั้งหมด (Schedules, Rules, RuleAttributes, Conditions)

---

## Constraints

- ลบได้เฉพาะ `status = DRAFT` หรือ `INACTIVE` เท่านั้น
- **ห้ามลบ** หาก `status = ACTIVE` — ต้อง Deactivate ก่อน (ระบบ return 422)
- ใช้ **Soft Delete** ทั้งหมด — ตั้งค่า `DELETED_AT = NOW()` ไม่มีการลบข้อมูลจริงออกจาก DB
- ลบ Child records ในลำดับที่ถูกต้องเพื่อหลีกเลี่ยงปัญหา FK constraint:
  1. `schedules`
  2. `rule_attributes`
  3. `rules`
  4. `rule_conditions`
  5. `decision_rules`
- ทุก operation อยู่ใน **single database transaction** (Commit หรือ Rollback ทั้งหมด)
- `UPDATED_BY` ถูก stamp โดยอัตโนมัติผ่าน GORM `BeforeDelete` hook ในทุก table

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Request Body

ไม่มี Request Body

---

## Response Body `204 No Content`

ไม่มี Response Body

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| `status = ACTIVE` | 422 | `ACTIVE decision rules cannot be deleted; deactivate first` |

---

## State Guard

```
DRAFT    ──✓──▶  Deleted   (อนุญาต)
INACTIVE ──✓──▶  Deleted   (อนุญาต)
ACTIVE   ──✗──▶  Deleted   (ไม่อนุญาต — Deactivate ก่อน)
```

---

## Transaction Flow (within single DB transaction)

```
BEGIN TRANSACTION

1. DELETE schedules
   WHERE DECISION_RULE_ID = {id}
   → soft-delete: SET DELETED_AT = NOW()

2. SELECT ID FROM rules
   WHERE DECISION_RULE_ID = {id}
   → รวบรวม rule_ids

3. ถ้า rule_ids ไม่ว่าง:
   a. DELETE rule_attributes WHERE RULE_ID IN (rule_ids)
   b. DELETE rules WHERE ID IN (rule_ids)

4. DELETE rule_conditions
   WHERE DECISION_RULE_ID = {id}

5. DELETE decision_rules
   WHERE ID = {id}

COMMIT
```

---

## DB Mapping

| Table | Filter | Column Set | Notes |
|-------|--------|-----------|-------|
| `schedules` | `DECISION_RULE_ID = {id}` | `DELETED_AT`, `UPDATED_BY`, `UPDATED_AT` | soft-delete |
| `rule_attributes` | `RULE_ID IN (rule_ids)` | `DELETED_AT`, `UPDATED_BY`, `UPDATED_AT` | soft-delete |
| `rules` | `ID IN (rule_ids)` | `DELETED_AT`, `UPDATED_BY`, `UPDATED_AT` | soft-delete |
| `rule_conditions` | `DECISION_RULE_ID = {id}` | `DELETED_AT`, `UPDATED_BY`, `UPDATED_AT` | soft-delete |
| `decision_rules` | `ID = {id}` | `DELETED_AT`, `UPDATED_BY`, `UPDATED_AT` | soft-delete |

> **หมายเหตุ:** GORM's `BeforeDelete` hook stamp `UPDATED_BY` โดยอัตโนมัติจาก context ในทุก soft-delete operation
