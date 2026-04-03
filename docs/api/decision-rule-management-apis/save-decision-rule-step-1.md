POST : /decision-rule/save/step-1

## Constraints

- Step 1st: Decision rule info (type, name, content path, score) and condition groups are sent together in one request.
- Max nesting depth: 3 levels
- Max conditions per group: 10 items
- `connectorOperator` (AND/OR) is on each condition, connecting it to the next item in sequence. The last item in a group has `connectorOperator: null`.
- Nested groups appear as items in `conditions` array with `"type": "nestedGroup"`.
- `value` format: raw value — backend infers structure from `logicalOperator`.

## Request Body

```json
{
  "decisionType": "SCORING",
  "decisionRuleName": "Gold Customer Rule",
  "contentPath": "/content/promo-banner",
  "score": 80,
  "conditionGroups": [
    {
      "conditions": [
        {
          "type": "condition",
          "source": "user_attribute",
          "attributeId": "uuid-attr-1",
          "logicalOperator": "=",
          "sequence": 1,
          "connectorOperator": "OR"
        },
        {
          "type": "condition",
          "source": "event_attribute",
          "attributeId": "uuid-attr-2",
          "logicalOperator": ">",
          "sequence": 2,
          "connectorOperator": "AND"
        },
        {
          "type": "nestedGroup",
          "sequence": 3,
          "connectorOperator": null,
          "conditionGroup": [
            {
              "conditions": [
                {
                  "type": "condition",
                  "source": "user_attribute",
                  "attributeId": "uuid-attr-3",
                  "logicalOperator": "IN",
                  "sequence": 1,
                  "connectorOperator": "OR"
                },
                {
                  "type": "condition",
                  "source": "event_attribute",
                  "attributeId": "uuid-attr-4",
                  "logicalOperator": "BETWEEN",
                  "sequence": 2,
                  "connectorOperator": null
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

## DB Mapping (Backend)

| JSON Path                              | Table                   | Notes                                                                                         |
| -------------------------------------- | ----------------------- | --------------------------------------------------------------------------------------------- |
| root `{}`                              | `decision_rules`        | decisionType→type, decisionRuleName→name, contentPath→content_path, score→score, status=DRAFT |
| `conditionGroups[n]`                   | `rule_condition_groups` | parent_rule_condition_groups=null for root, rule_operator field removed or ignored            |
| `conditions[n]` where type=condition   | `rule_condition`        | connectorOperator stored as new column                                                        |
| `conditions[n]` where type=nestedGroup | `rule_condition_groups` | parent = parent group's id, recursive                                                         |

## Model Changes Required

1. `rules.decision_rule_id` — **uncomment** FK to `decision_rules.id`
2. `rule_condition` — **add** `connector_operator enum` (AND, OR, nullable)
3. `rule_condition_groups.rule_operator` — **optional/remove**, group is visual container only
