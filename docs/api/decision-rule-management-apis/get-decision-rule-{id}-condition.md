# GET /decision-rules/{id}/conditions

ดึงข้อมูล `DecisionRule` และ `RuleCondition` tree ของ Wizard Step 1 (ใช้สำหรับโหมด Edit)

---

## Constraints

- Response มี structure เดียวกับ Request ของ Step 1 เพื่อให้ frontend populate form ได้โดยตรง
- `type: "group"` rows จะมี `conditionId` เพื่อใช้ใน edit/delete operations
- `attributeIsActive: false` → frontend แสดง warning badge บน condition นั้น (warn only, ไม่ block save)
- Query ดึง `rule_conditions` เฉพาะที่เป็น **template** (`RULE_ID IS NULL`) — ไม่รวม condition values จาก Step 2

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Response Body `200 OK`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleId": "DR-20260424-001",
    "type": "AUDIENCE",
    "evaluateType": "SCORING",
    "name": "Gold Customer Rule",
    "contentPath": "/content/promo-banner",
    "campaignCode": "CAMP2026Q2",
    "score": 80,
    "status": "DRAFT",
    "subStatus": "N/A",
    "conditions": [
      {
        "conditionId": "c1000000-0000-0000-0000-000000000001",
        "type": "condition",
        "sequence": 1,
        "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3302",
        "attributeName": "Customer Segment",
        "attributeIsActive": true,
        "logicalOperator": "IN",
        "connectorOperator": "OR"
      },
      {
        "conditionId": "c1000000-0000-0000-0000-000000000002",
        "type": "condition",
        "sequence": 2,
        "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3303",
        "attributeName": "Account Balance",
        "attributeIsActive": false,
        "logicalOperator": ">",
        "connectorOperator": "AND"
      },
      {
        "conditionId": "c1000000-0000-0000-0000-000000000003",
        "type": "group",
        "sequence": 3,
        "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3304",
        "attributeName": "Transaction Group",
        "attributeIsActive": true,
        "logicalOperator": "=",
        "connectorOperator": null,
        "conditions": [
          {
            "conditionId": "c1000000-0000-0000-0000-000000000004",
            "type": "condition",
            "sequence": 1,
            "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3305",
            "attributeName": "Transaction Count",
            "attributeIsActive": true,
            "logicalOperator": ">=",
            "connectorOperator": "OR"
          },
          {
            "conditionId": "c1000000-0000-0000-0000-000000000005",
            "type": "condition",
            "sequence": 2,
            "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3306",
            "attributeName": "Last Transaction Date",
            "attributeIsActive": true,
            "logicalOperator": "BETWEEN",
            "connectorOperator": null
          }
        ]
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
| `type` | `decision_rules.type` | DecisionType enum |
| `evaluateType` | `decision_rules.evaluate_type` | EvaluateType enum |
| `status` | `decision_rules.status` | DRAFT / ACTIVE / INACTIVE |
| `subStatus` | `decision_rules.sub_status` | N/A / Missing attribute registry |
| `conditions[].conditionId` | `rule_conditions.id` | UUID ใช้อ้างอิงใน Step 2 |
| `conditions[].type` | derived | `"group"` เมื่อ `attribute_id IS NULL` |
| `conditions[].attributeName` | `attributes.display_name` | JOIN via `attribute_id` |
| `conditions[].attributeIsActive` | `attributes.is_active` | |

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |

---

## Query Strategy

```sql
-- 1. decision rule metadata
SELECT id, decision_rule_id, type, evaluate_type, name, content_path,
       campaign_code, score, status, sub_status, created_at, updated_at
FROM decision_rules
WHERE id = :id AND deleted_at IS NULL;

-- 2. template conditions พร้อม attribute info (flat → tree ใน application layer)
SELECT
  rc.id           AS condition_id,
  rc.sequence,
  rc.parent_rule_condition_id,
  rc.attribute_id,
  rc.logical_operator,
  rc.connector_operator,
  a.display_name  AS attribute_name,
  a.is_active     AS attribute_is_active
FROM rule_conditions rc
LEFT JOIN attributes a ON rc.attribute_id = a.id AND a.deleted_at IS NULL
WHERE rc.decision_rule_id = :id
  AND rc.rule_id IS NULL
  AND rc.deleted_at IS NULL
ORDER BY rc.parent_rule_condition_id NULLS FIRST, rc.sequence;
```

**หมายเหตุ:**
- `LEFT JOIN` เพราะ `type: "group"` rows มี `attribute_id = null`
- `rc.rule_id IS NULL` กรองเฉพาะ template conditions (Step 2 rows จะมี `rule_id` set)
- Application layer สร้าง recursive tree จาก flat rows โดยใช้ `parent_rule_condition_id`
