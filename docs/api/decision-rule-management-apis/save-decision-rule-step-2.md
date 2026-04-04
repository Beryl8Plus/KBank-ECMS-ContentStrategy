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
