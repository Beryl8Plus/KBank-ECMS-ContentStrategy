# PUT /decision-rules/{id}/rule-sets

บันทึก `Rule` และ `RuleAttribute` values (Wizard Step 2 — write)

---

## Constraints

- `ruleId: null` → INSERT `rules` row ใหม่ + INSERT `rule_attributes`
- `ruleId: non-null` → UPDATE `rules` row + UPSERT `rule_attributes` (by `rule_id` + `ruleConditionId`)
- `conditionId` ต้องอ้างอิง `rule_conditions.id` ที่ `decision_rule_id = {id}` และ `rule_id IS NULL` (template)
- `orderNo` ต้องไม่ซ้ำกันภายใน request เดียวกัน
- Backend validate ว่า `conditionId` ทุกตัวที่ส่งมา belong to decision rule นี้จริง
- Partial values อนุญาต — `value: null` หมายถึงยังไม่กรอก ไม่ใช่ error
- Request นี้ไม่เปลี่ยน `decision_rules.status`

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

| Case | HTTP | Code | Message |
|------|------|------|---------|
| `id` ไม่มีใน DB | 404 | `NOT_FOUND` | `decision rule not found` |
| `conditionId` ไม่ belong to decision rule | 422 | `VALIDATION_ERROR` | `condition {id} does not belong to this decision rule` |
| `ruleId` non-null แต่ไม่พบใน DB | 404 | `NOT_FOUND` | `rule {id} not found` |
| `orderNo` ซ้ำใน request | 422 | `VALIDATION_ERROR` | `duplicate orderNo {n} in ruleSets` |
| `ruleSets` ว่าง | 400 | `INVALID_FIELD` | `ruleSets must contain at least one item` |

---

## DB Mapping

| JSON Path | Table | Column | Notes |
|-----------|-------|--------|-------|
| `{id}` (path) | `decision_rules` | `ID` | validate exists |
| `ruleSets[n]` (ruleId=null) | `rules` | `DECISION_RULE_ID`, `VARIATION_NAME`, `SCORE`, `ORDER_NO` | INSERT |
| `ruleSets[n]` (ruleId=set) | `rules` | `VARIATION_NAME`, `SCORE`, `ORDER_NO` | UPDATE WHERE `ID = ruleId` |
| `ruleSets[n].conditions[m]` | `rule_attributes` | `RULE_ID`, `ATTRIBUTE_ID`, `RULE_CONDITION_ID`, `VALUE` | UPSERT |

### Rule Attribute Upsert Logic

```
for each ruleSet:
  1. CREATE or UPDATE rules row → get rule.id
  2. lookup attribute_id from rule_conditions WHERE id = conditionId
  3. UPSERT rule_attributes
     ON CONFLICT (rule_id, rule_condition_id) DO UPDATE SET value = excluded.value
```

> **Entity Change Required:** `rule_attributes` ต้องเพิ่ม column `RULE_CONDITION_ID UUID FK → rule_conditions.id`
> เพื่อให้ GET rule-sets สามารถ map value กลับไปยัง column ได้ถูกต้อง
> และรองรับกรณีที่ attribute เดียวกันถูกใช้มากกว่า 1 condition ใน template
