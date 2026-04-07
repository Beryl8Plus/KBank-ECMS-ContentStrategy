POST : /decision-rule/save/step-1

## Constraints

- Step 1st: Decision rule info (type, name, content path, score) and conditions are sent together in one request.
- Max nesting depth: 3 levels
- Max conditions per group: 10 items
- `connectorOperator` (AND/OR) is on each item, connecting it to the next item in sequence. The last item has `connectorOperator: null`.
- `type: "group"` items contain a nested `conditions` array (recursive structure).
- `type: "group"` rows are stored as `rule_condition` with `attribute_id = null`, `logical_operator = null`, `value = null`.
- `value` format: raw value — backend infers structure from `logicalOperator`.

## Request Body

```json
{
  "decisionType": "SCORING",
  "decisionRuleName": "Gold Customer Rule",
  "contentPath": "/content/promo-banner",
  "score": 80,
  "conditions": [
    {
      "type": "group",
      "sequence": 1,
      "source": "user_attribute",
      "attributeId": "uuid-attr-group-1",
      "logicalOperator": "=",
      "connectorOperator": "OR",
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
          "source": "user_attribute",
          "attributeId": "uuid-attr-5",
          "logicalOperator": "!=",
          "sequence": 2,
          "connectorOperator": null
        }
      ]
    },
    {
      "type": "group",
      "sequence": 2,
      "source": "event_attribute",
      "attributeId": "uuid-attr-group-2",
      "logicalOperator": "=",
      "connectorOperator": "AND",
      "conditions": [
        {
          "type": "condition",
          "source": "event_attribute",
          "attributeId": "uuid-attr-2",
          "logicalOperator": ">",
          "sequence": 1,
          "connectorOperator": "OR"
        },
        {
          "type": "group",
          "sequence": 2,
          "source": "event_attribute",
          "attributeId": "uuid-attr-group-3",
          "logicalOperator": "=",
          "connectorOperator": null,
          "conditions": [
            {
              "type": "condition",
              "source": "user_attribute",
              "attributeId": "uuid-attr-6",
              "logicalOperator": "<=",
              "sequence": 1,
              "connectorOperator": "OR"
            },
            {
              "type": "condition",
              "source": "user_attribute",
              "attributeId": "uuid-attr-7",
              "logicalOperator": "IN",
              "sequence": 2,
              "connectorOperator": null
            }
          ]
        }
      ]
    },
    {
      "type": "group",
      "sequence": 3,
      "attributeId": "uuid-attr-group-4",
      "source": "user_attribute",
      "logicalOperator": "=",
      "connectorOperator": "AND",
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
```

## DB Mapping (Backend)

| JSON Path                            | Table            | Notes                                                                                         |
| ------------------------------------ | ---------------- | --------------------------------------------------------------------------------------------- |
| root `{}`                            | `decision_rules` | decisionType→type, decisionRuleName→name, contentPath→content_path, score→score, status=DRAFT |
| `conditions[n]` where type=condition | `rule_condition` | `parent_rule_condition_id = null` for root-level, connectorOperator→connector_operator        |
| `conditions[n]` where type=group     | `rule_condition` | `attribute_id = null`, `logical_operator = null`, `value = null` — group container row        |
| nested `conditions[n]`               | `rule_condition` | `parent_rule_condition_id` = parent group row's id, recursive                                 |
