# POST /decision-rules

สร้าง Draft `DecisionRule` พร้อม `RuleCondition` (Wizard Step 1)

---

## Constraints

- สร้าง `DecisionRule` ด้วย `status = DRAFT` ทันที
- Max nesting depth: **3 ระดับ**
- Max conditions ต่อ group: **10 items**
- `type: "group"` ต้องมี `conditions` array ที่ไม่ว่าง
- `type: "condition"` ต้องมี `attributeId`, `logicalOperator`
- `connectorOperator` ของ item **สุดท้าย** ใน array ต้องเป็น `null`
- `type: "group"` rows ถูกบันทึกใน `rule_conditions` โดยมี `attribute_id = null`, `logical_operator = null`
- `campaignCode` จำเป็นเฉพาะเมื่อ `type` เป็น `AUDIENCE` หรือ `SALES_TARGET` (IsCampaign)

---

## Request Body

```json
{
  "type": "AUDIENCE",
  "evaluateType": "SCORING",
  "name": "Gold Customer Rule",
  "contentPath": "/content/promo-banner",
  "campaignCode": "CAMP2026Q2",
  "score": 80,
  "conditions": [
    {
      "type": "group",
      "sequence": 1,
      "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3301",
      "logicalOperator": "=",
      "connectorOperator": "OR",
      "conditions": [
        {
          "type": "condition",
          "sequence": 1,
          "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3302",
          "logicalOperator": "IN",
          "connectorOperator": "OR"
        },
        {
          "type": "condition",
          "sequence": 2,
          "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3303",
          "logicalOperator": "!=",
          "connectorOperator": null
        }
      ]
    },
    {
      "type": "group",
      "sequence": 2,
      "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3304",
      "logicalOperator": "=",
      "connectorOperator": "AND",
      "conditions": [
        {
          "type": "condition",
          "sequence": 1,
          "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3305",
          "logicalOperator": ">",
          "connectorOperator": "OR"
        },
        {
          "type": "group",
          "sequence": 2,
          "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3306",
          "logicalOperator": "=",
          "connectorOperator": null,
          "conditions": [
            {
              "type": "condition",
              "sequence": 1,
              "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3307",
              "logicalOperator": "<=",
              "connectorOperator": "OR"
            },
            {
              "type": "condition",
              "sequence": 2,
              "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3308",
              "logicalOperator": "BETWEEN",
              "connectorOperator": null
            }
          ]
        }
      ]
    }
  ]
}
```

### Field Reference

| Field | Type | Required | Valid Values | Notes |
|-------|------|----------|-------------|-------|
| `type` | string | ✓ | `MASS`, `AUDIENCE`, `SALES_TARGET`, `NON_SALES` | maps to `decision_rules.type` |
| `evaluateType` | string | ✓ | `SCORING`, `SEGMENT`, `ELIGIBLE` | maps to `decision_rules.evaluate_type` |
| `name` | string | ✓ | max 255 chars | |
| `contentPath` | string | ✓ | max 255 chars | |
| `campaignCode` | string | ✓ when IsCampaign | max 25 chars | required when type = AUDIENCE or SALES_TARGET |
| `score` | float | — | 0–100 | default 0 |
| `conditions[].type` | string | ✓ | `condition`, `group` | |
| `conditions[].sequence` | int | ✓ | ≥ 1 | ordering within parent |
| `conditions[].attributeId` | UUID | ✓ for condition | — | null allowed for group rows |
| `conditions[].logicalOperator` | string | ✓ for condition | `<`, `<=`, `>`, `>=`, `=`, `!=`, `IN`, `BETWEEN` | null for group rows |
| `conditions[].connectorOperator` | string | — | `AND`, `OR`, `null` | null on last item in array |
| `conditions[].conditions` | array | ✓ for group | — | recursive; required when type = group |

---

## Response Body `201 Created`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleId": "DR-20260424-001",
    "status": "DRAFT",
    "createdAt": "2026-04-24T10:00:00+07:00"
  }
}
```

| Field | Source | Notes |
|-------|--------|-------|
| `id` | `decision_rules.id` | UUID primary key (BaseModel) |
| `decisionRuleId` | `decision_rules.decision_rule_id` | business key; unique string |
| `status` | `decision_rules.status` | always `DRAFT` at creation |

---

## Validation Errors

| Case | HTTP | Code | Message |
|------|------|------|---------|
| `type` ไม่ถูกต้อง | 400 | `INVALID_FIELD` | `type must be one of MASS, AUDIENCE, SALES_TARGET, NON_SALES` |
| `evaluateType` ไม่ถูกต้อง | 400 | `INVALID_FIELD` | `evaluateType must be one of SCORING, SEGMENT, ELIGIBLE` |
| `campaignCode` หายไปเมื่อ IsCampaign | 400 | `INVALID_FIELD` | `campaignCode is required for AUDIENCE and SALES_TARGET types` |
| `attributeId` ไม่มีใน DB | 404 | `NOT_FOUND` | `attribute {id} not found` |
| nesting depth > 3 | 422 | `VALIDATION_ERROR` | `conditions exceed maximum nesting depth of 3` |
| conditions ต่อ group > 10 | 422 | `VALIDATION_ERROR` | `group exceeds maximum of 10 conditions` |
| `type=condition` แต่ `attributeId` เป็น null | 422 | `VALIDATION_ERROR` | `attributeId is required for condition type` |
| `type=group` แต่ `conditions` ว่าง | 422 | `VALIDATION_ERROR` | `group must contain at least one condition` |
| `connectorOperator` ของ item สุดท้ายไม่เป็น null | 422 | `VALIDATION_ERROR` | `last condition in array must have connectorOperator null` |
| ชื่อซ้ำใน system | 409 | `CONFLICT` | `decision rule with this name already exists` |

---

## DB Mapping

| JSON Field | Table | Column | Notes |
|------------|-------|--------|-------|
| `type` | `decision_rules` | `TYPE` | DecisionType enum |
| `evaluateType` | `decision_rules` | `EVALUATE_TYPE` | EvaluateType enum |
| `name` | `decision_rules` | `NAME` | |
| `contentPath` | `decision_rules` | `CONTENT_PATH` | |
| `campaignCode` | `decision_rules` | `CAMPAIGN_CODE` | |
| `score` | `decision_rules` | `SCORE` | |
| — | `decision_rules` | `STATUS` | hardcoded `DRAFT` |
| `conditions[n]` (type=condition) | `rule_conditions` | — | `PARENT_RULE_CONDITION_ID = null` for root-level |
| `conditions[n]` (type=group) | `rule_conditions` | — | `ATTRIBUTE_ID = null`, `LOGICAL_OPERATOR = null` |
| nested `conditions[n]` | `rule_conditions` | `PARENT_RULE_CONDITION_ID` | FK → parent group row's `ID` |
