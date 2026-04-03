# POST : /decision-rule/save/step-3

## Constraints

- Step 3rd: Set schedule for activating the decision rule.
- Frontend sends `decisionRuleId` and an array of `schedules`, each containing `placementId`, `startDate`, and `endDate`.
- **Maximum 3 schedules per placement** — backend must validate that the total number of existing schedules (across all decision rules) for a given `placementId` does not exceed 3 after the operation.
- Each `(decisionRuleId, placementId)` pair must be unique — inserting a duplicate must return a validation error.
- `startDate` must be before `endDate`.
- `decisionRuleId` must reference an existing `decision_rules` record.
- `placementId` is required and must not be duplicated for the same `decisionRuleId`. Backend validates no overlapping schedule for the same `placementId`.
- `placementId` must reference an existing `placements` record.
- After saving, backend sets `decision_rules.status` to `ACTIVE`.

## Request Body

```json
{
  "decisionRuleId": "uuid-decision-rule",
  "schedules": [
    {
      "placementId": "uuid-placement-1",
      "startDate": "2026-05-01T00:00:00+07:00",
      "endDate": "2026-05-31T23:59:59+07:00"
    },
    {
      "placementId": "uuid-placement-2",
      "startDate": "2026-06-01T00:00:00+07:00",
      "endDate": "2026-06-30T23:59:59+07:00"
    }
  ]
}
```

## Response Body

```json
{
  "decisionRuleId": "uuid-decision-rule",
  "schedules": [
    {
      "scheduleId": "uuid-schedule-1",
      "placementId": "uuid-placement-1",
      "startDate": "2026-05-01T00:00:00+07:00",
      "endDate": "2026-05-31T23:59:59+07:00",
      "isActive": true
    },
    {
      "scheduleId": "uuid-schedule-2",
      "placementId": "uuid-placement-2",
      "startDate": "2026-06-01T00:00:00+07:00",
      "endDate": "2026-06-30T23:59:59+07:00",
      "isActive": true
    }
  ]
}
```

## Validation Errors

| Case                                      | HTTP Status | Message                                                        |
| ----------------------------------------- | ----------- | -------------------------------------------------------------- |
| `decisionRuleId` not found                | 404         | `decision rule not found`                                      |
| `placementId` not found                   | 404         | `placement not found`                                          |
| `startDate` >= `endDate`                  | 400         | `startDate must be before endDate`                             |
| placement already has 3 schedules         | 422         | `placement has reached the maximum of 3 schedules`             |
| duplicate `(decisionRuleId, placementId)` | 422         | `schedule for this decision rule and placement already exists` |

## DB Mapping (Backend)

| JSON Path                  | Table            | Column                                                                              | Notes                                                   |
| -------------------------- | ---------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `decisionRuleId`           | `decision_rules` | `id`                                                                                | Reference — validate exists; update `status` → `ACTIVE` |
| `schedules[n].placementId` | `placements`     | `id`                                                                                | Reference — validate exists                             |
| `schedules[n]`             | `schedules`      | `decision_rule_id`, `placement_id`, `start_timestamp`, `end_timestamp`, `is_active` | INSERT new record; `is_active` defaults to `true`       |

### Capacity Check (per placement)

Before inserting, backend runs:

```sql
SELECT COUNT(*) FROM schedules
WHERE placement_id = :placementId;
```

If count >= 3, reject with 422.
