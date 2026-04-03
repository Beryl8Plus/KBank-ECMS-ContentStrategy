# GET : /decision-rule/{id}/rule-sets

## Constraints

- Step 2nd: After saving the basic info and condition groups in step 1, frontend fetches rule-sets to render the rule table.
- `columns` — derived from conditions saved in step 1 (attribute + operator per column).
- `ruleSets` — existing rule rows; empty array if no rules exist yet.
- `ruleId: null` in POST means "create new rule row"; non-null means "update existing".

## Response Body

```json
{
  "decisionRuleId": "uuid",
  "columns": [
    {
      "conditionId": "uuid",
      "attributeId": "uuid",
      "attributeName": "Attribute 1, Sandbox",
      "logicalOperator": "=",
      "dataType": "Text"
    },
    {
      "conditionId": "uuid",
      "attributeId": "uuid",
      "attributeName": "Attribute 2, Sandbox",
      "logicalOperator": "=",
      "dataType": "Text"
    },
    {
      "conditionId": "uuid",
      "attributeId": "uuid",
      "attributeName": "Attribute 3, Sandbox",
      "logicalOperator": ">",
      "dataType": "Number"
    }
  ],
  "ruleSets": [
    {
      "ruleId": "uuid",
      "orderNo": 1,
      "score": 80,
      "variation": "HNW Wealth",
      "values": [
        { "conditionId": "uuid", "value": "Attribute Value" },
        { "conditionId": "uuid", "value": "Attribute Value" },
        { "conditionId": "uuid", "value": null }
      ]
    },
    {
      "ruleId": "uuid",
      "orderNo": 2,
      "score": null,
      "variation": null,
      "values": [
        { "conditionId": "uuid", "value": "Attribute Value" },
        { "conditionId": "uuid", "value": null },
        { "conditionId": "uuid", "value": null }
      ]
    }
  ]
}
```

## GET DB Mapping (Backend)

| JSON Path               | DB Table                        | Notes                                                                         |
| ----------------------- | ------------------------------- | ----------------------------------------------------------------------------- |
| `decisionRuleId`        | `decision_rules`                | id                                                                            |
| `columns[n]`            | `rule_condition` → `attributes` | conditionId = rule_condition.id, join attributes for display_name & data_type |
| `ruleSets[n]`           | `rules`                         | ruleId, variation_name, score, order_no                                       |
| `ruleSets[n].values[m]` | `rule_condition`                | value (jsonb) filtered by rule_id                                             |

---

# POST : /decision-rule/save/step-2

## Constraints

- Step 2nd: Frontend sends the rule rows with attribute values filled in the table.
- `ruleId: null` = create new `rules` record; non-null = update existing.
- `conditions[n].conditionId` references `rule_condition.id` from step 1.
- Backend sets `rule_condition.rule_id` to link condition values to their rule row.

## Request Body

```json
{
  "decisionRuleId": "uuid",
  "ruleSets": [
    {
      "ruleId": "uuid-or-null",
      "orderNo": 1,
      "score": 80,
      "variation": "HNW Wealth",
      "conditions": [
        { "conditionId": "uuid", "value": "Attribute Value" },
        { "conditionId": "uuid", "value": "Attribute Value" },
        { "conditionId": "uuid", "value": null }
      ]
    },
    {
      "ruleId": null,
      "orderNo": 2,
      "score": null,
      "variation": "HNW non-Wealth",
      "conditions": [
        { "conditionId": "uuid", "value": null },
        { "conditionId": "uuid", "value": null },
        { "conditionId": "uuid", "value": null }
      ]
    }
  ]
}
```

## POST DB Mapping (Backend)

| JSON Path                   | DB Table         | Column                                            | Notes                                                             |
| --------------------------- | ---------------- | ------------------------------------------------- | ----------------------------------------------------------------- |
| `decisionRuleId`            | `decision_rules` | id                                                | Reference — validate exists                                       |
| `ruleSets[n]`               | `rules`          | decision_rule_id, variation_name, score, order_no | ruleId=null → INSERT, non-null → UPDATE                           |
| `ruleSets[n].conditions[m]` | `rule_condition` | value, rule_id                                    | UPDATE value WHERE id=conditionId; SET rule_id to the parent rule |
