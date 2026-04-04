GET /decision-rules/{decision_rule_id}/conditions

## Description

Get the conditions of a specific decision rule. Used when returning to edit step 1 — response mirrors POST step 1 structure so the form can be populated directly.

- Each condition includes `attributeIsActive` from `attributes.is_active` for frontend validation warning.
- If `attributeIsActive: false`, frontend should display a warning badge on that condition (warn only, do not block save).
- `type: "group"` items include `conditionId` (the group row's `rule_condition.id`) for edit/delete operations.

## Response Body

```json
{
  "decisionRuleId": "uuid",
  "decisionType": "SCORING",
  "decisionRuleName": "Gold Customer Rule",
  "contentPath": "/content/promo-banner",
  "score": 80,
  "status": "DRAFT",
  "conditions": [
    {
      "conditionId": "uuid-cond-1",
      "type": "condition",
      "source": "user_attribute",
      "attributeId": "uuid-attr-1",
      "attributeIsActive": true,
      "logicalOperator": "=",
      "sequence": 1,
      "connectorOperator": "OR"
    },
    {
      "conditionId": "uuid-cond-2",
      "type": "condition",
      "source": "event_attribute",
      "attributeId": "uuid-attr-2",
      "attributeIsActive": false,
      "logicalOperator": ">",
      "sequence": 2,
      "connectorOperator": "AND"
    },
    {
      "conditionId": "uuid-group-1",
      "type": "group",
      "sequence": 3,
      "attributeId": "uuid-attr-group-4",
      "attributeIsActive": true,
      "logicalOperator": "=",
      "connectorOperator": "AND",
      "conditions": [
        {
          "conditionId": "uuid-cond-3",
          "type": "condition",
          "source": "user_attribute",
          "attributeId": "uuid-attr-3",
          "attributeIsActive": true,
          "logicalOperator": "IN",
          "sequence": 1,
          "connectorOperator": "OR"
        },
        {
          "conditionId": "uuid-cond-4",
          "type": "condition",
          "source": "event_attribute",
          "attributeId": "uuid-attr-4",
          "attributeIsActive": true,
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

| JSON Path | DB Table | Notes |
|---|---|---|
| `decisionRuleId`, `decisionType`, `decisionRuleName`, `contentPath`, `score`, `status` | `decision_rules` | id, type, name, content_path, score, status |
| `conditions[n]` where type=condition | `rule_condition` | conditionId = rule_condition.id, `parent_rule_condition_id IS NULL` for root |
| `conditions[n].attributeIsActive` | `attributes` | JOIN `attributes ON rule_condition.attribute_id = attributes.id` → `attributes.is_active` |
| `conditions[n].source` | `attributes` | `attributes.source_system` |
| `conditions[n]` where type=group | `rule_condition` | `attribute_id IS NULL` — group container row, conditionId = rule_condition.id |
| nested `conditions[n]` | `rule_condition` | `parent_rule_condition_id = parent group's id`, recursive |

## Query Strategy

```sql
-- 1. Get decision rule metadata
SELECT id, type, name, content_path, score, status
FROM decision_rules
WHERE id = :decision_rule_id;

-- 2. Get all conditions (build tree in application layer using parent_rule_condition_id)
SELECT
  rc.id AS condition_id,
  rc.sequence,
  rc.parent_rule_condition_id,
  rc.attribute_id,
  rc.logical_operator,
  rc.connector_operator,
  a.source_system AS source,
  a.is_active AS attribute_is_active
FROM rule_condition rc
LEFT JOIN attributes a ON rc.attribute_id = a.id
WHERE rc.decision_rule_id = :decision_rule_id
  AND rc.rule_id IS NULL
ORDER BY rc.parent_rule_condition_id NULLS FIRST, rc.sequence;
```

**Notes:**
- `LEFT JOIN` instead of `JOIN` because `type: "group"` rows have `attribute_id = null`.
- `rc.rule_id IS NULL` filters to step 1 conditions only (step 2 conditions have `rule_id` set).
- Application layer builds the recursive tree from flat rows using `parent_rule_condition_id`.
