# PUT /decision-rules/{id}/rule-sets

บันทึก `Rule` และ `RuleAttribute` values (Wizard Step 2 — write)

---

## Constraints

- Endpoint นี้ใช้ได้ทั้ง **Create Mode** และ **Edit Mode** (same endpoint, same logic)
- `ruleId: null` → **INSERT** `rules` row ใหม่ + INSERT `rule_attributes`
- `ruleId: non-null` → **UPDATE** `rules` row + full-replace `rule_attributes` สำหรับ rule นั้น
- **Rule ที่ไม่อยู่ใน request จะถูก DELETE** — ทั้ง `rules` row และ `rule_attributes` ของมัน (delta delete)
- `conditionId` ต้องอ้างอิง `rule_conditions.id` ที่ `decision_rule_id = {id}`
- `orderNo` ต้องไม่ซ้ำกันภายใน request เดียวกัน
- Backend validate ว่า `conditionId` ทุกตัวที่ส่งมา belong to decision rule นี้จริง
- Partial values อนุญาต — `value: null` หมายถึงยังไม่กรอก ไม่ใช่ error
- **Value Validation:** สำหรับ Attribute ที่มี `value` เป็น JSON options array — ค่าที่กรอก (`value` ที่ไม่ใช่ null) จะต้องอยู่ใน allowed set นั้น ถ้าไม่อยู่จะถูก reject ทันที (ป้องกันการบันทึก value ที่ถูกถอดออกจาก schema)
- Request นี้ไม่เปลี่ยน `decision_rules.status`
- ทุก operation ทำภายใน **single database transaction**

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Request Body

```json
{
  "ruleSets": [
    {
      "ruleId": null,
      "orderNo": 1,
      "score": 80,
      "variationName": "HNW Wealth",
      "conditions": [
        { "conditionId": "c1000000-0000-0000-0000-000000000001", "value": "Gold" },
        { "conditionId": "c1000000-0000-0000-0000-000000000002", "value": "500000" },
        { "conditionId": "c1000000-0000-0000-0000-000000000004", "value": null }
      ]
    },
    {
      "ruleId": "r1000000-0000-0000-0000-000000000001",
      "orderNo": 2,
      "score": 60,
      "variationName": "HNW Non-Wealth",
      "conditions": [
        { "conditionId": "c1000000-0000-0000-0000-000000000001", "value": "Silver" },
        { "conditionId": "c1000000-0000-0000-0000-000000000002", "value": null },
        { "conditionId": "c1000000-0000-0000-0000-000000000004", "value": "10" }
      ]
    }
  ]
}
```

### Field Reference

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `ruleSets` | array | ✓ | min 1 item |
| `ruleSets[].ruleId` | UUID \| null | — | null = create new rule |
| `ruleSets[].orderNo` | int | ✓ | ≥ 1; unique within request |
| `ruleSets[].score` | float | — | 0–100; null allowed |
| `ruleSets[].variationName` | string | ✓ | max 255 chars; maps to `rules.variation_name` |
| `ruleSets[].conditions` | array | ✓ | — |
| `ruleSets[].conditions[].conditionId` | UUID | ✓ | FK → `rule_conditions.id` (template) |
| `ruleSets[].conditions[].value` | string \| null | — | raw string; null = no value yet |

---

## Response Body `200 OK`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "savedRuleSets": 2,
    "updatedAt": "2026-04-24T10:01:00+07:00"
  }
}
```

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| `conditionId` ไม่ belong to decision rule | 422 | `condition {id} does not belong to this decision rule` |
| `ruleId` non-null แต่ไม่พบใน DB | 404 | `rule {id} not found` |
| `orderNo` ซ้ำใน request | 422 | `duplicate orderNo {n} in ruleSets` |
| `ruleSets` ว่าง | 400 | `ruleSets must contain at least one item` |
| `value` ไม่อยู่ใน allowed options ของ Attribute | 422 | `attribute value no longer in allowed options: rule_attribute {id}: value "{val}" not in allowed set [...]` |

---

## DB Mapping

| JSON Path | Table | Column | Operation |
|-----------|-------|--------|-----------|
| `{id}` (path) | `decision_rules` | `ID` | validate exists |
| rules in DB ที่ไม่อยู่ใน request | `rule_attributes` | `DELETED_AT` | soft-DELETE ก่อน |
| rules in DB ที่ไม่อยู่ใน request | `rules` | `DELETED_AT` | soft-DELETE |
| `ruleSets[n]` (ruleId=null) | `rules` | `DECISION_RULE_ID`, `VARIATION_NAME`, `SCORE`, `ORDER_NO` | INSERT |
| `ruleSets[n]` (ruleId=set) | `rules` | `VARIATION_NAME`, `SCORE`, `ORDER_NO` | UPDATE WHERE `ID = ruleId` |
| `ruleSets[n].conditions[m]` (value≠null) | `rule_attributes` | `RULE_ID`, `ATTRIBUTE_ID`, `VALUE` | DELETE existing + INSERT new |

### Transaction Logic

```
BEGIN TRANSACTION

1. ดึง existing rule IDs สำหรับ DR นี้
2. คำนวณ to_delete_rule_ids = existing − incoming
3. DELETE rule_attributes WHERE RULE_ID IN (to_delete_rule_ids)
4. DELETE rules WHERE ID IN (to_delete_rule_ids)
5. for each ruleSet:
   a. INSERT or UPDATE rules row
   b. DELETE rule_attributes WHERE RULE_ID = rule.id  (full replace)
   c. INSERT rule_attributes for each non-null value
      (attribute_id ได้จาก rule_conditions WHERE id = conditionId)

COMMIT
```

> **Attribute Lookup:** `conditionId` → `rule_conditions.attribute_id` → `rule_attributes.attribute_id`
> เนื่องจาก Option A บังคับว่า attribute แต่ละตัวใช้ได้ครั้งเดียวใน DR
> การ map ผ่าน attribute_id จึงถูกต้องและไม่ ambiguous
