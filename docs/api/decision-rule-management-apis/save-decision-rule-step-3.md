# PUT /decision-rules/{id}/schedules

กำหนด `Schedule` และ `Placement` สำหรับ Decision Rule (Wizard Step 3)

หลังจาก save สำเร็จ backend จะเปลี่ยน `decision_rules.status` → `ACTIVE`

---

## Constraints

- Max **3 schedules ต่อ placement** (นับรวมทุก decision rules ใน system)
- แต่ละ `(decision_rule_id, placement_id)` pair ต้องไม่ซ้ำ
- `startDate` ต้องน้อยกว่า `endDate`
- `endDate` ต้องไม่อยู่ในอดีต
- `placementId` ต้องไม่ซ้ำภายใน request เดียวกัน
- `timezone` default เป็น `Asia/Bangkok` หาก frontend ไม่ส่งมา
- `timeOfDayStart` / `timeOfDayEnd` format: `HH:mm` (24-hour) เช่น `"08:00"`, `"23:59"`
- `allDay: true` → ไม่ต้องส่ง `timeOfDayStart` / `timeOfDayEnd`
- `recurrenceType: "RRULE"` → ต้องส่ง `recurrenceRule` (iCalendar RRULE format)
- `recurrenceType: "CALENDAR"` → ต้องส่ง `calendarId`
- `recurrenceType: "ONCE"` → ทำงานครั้งเดียวตาม `startDate`–`endDate`

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Request Body

**Simple (ONCE, AllDay):**
```json
{
  "schedules": [
    {
      "placementId": "p1000000-0000-0000-0000-000000000001",
      "startDate": "2026-05-01T00:00:00+07:00",
      "endDate": "2026-05-31T23:59:59+07:00",
      "recurrenceType": "ONCE",
      "allDay": true,
      "timezone": "Asia/Bangkok"
    }
  ]
}
```

**Advanced (RRULE, Time window):**
```json
{
  "schedules": [
    {
      "placementId": "p1000000-0000-0000-0000-000000000001",
      "startDate": "2026-05-01T00:00:00+07:00",
      "endDate": "2026-05-31T23:59:59+07:00",
      "recurrenceType": "RRULE",
      "recurrenceRule": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
      "allDay": false,
      "timeOfDayStart": "08:00",
      "timeOfDayEnd": "18:00",
      "timezone": "Asia/Bangkok"
    },
    {
      "placementId": "p1000000-0000-0000-0000-000000000002",
      "startDate": "2026-06-01T00:00:00+07:00",
      "endDate": "2026-06-30T23:59:59+07:00",
      "recurrenceType": "CALENDAR",
      "calendarId": "ca100000-0000-0000-0000-000000000001",
      "allDay": true,
      "timezone": "Asia/Bangkok"
    }
  ]
}
```

### Field Reference

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `schedules` | array | ✓ | min 1 item |
| `schedules[].placementId` | UUID | ✓ | FK → `placements.id` |
| `schedules[].startDate` | RFC3339 | ✓ | stored as UTC in `effective_from` |
| `schedules[].endDate` | RFC3339 | ✓ | stored as UTC in `effective_until` |
| `schedules[].recurrenceType` | string | ✓ | `ONCE`, `RRULE`, `CALENDAR` |
| `schedules[].recurrenceRule` | string | ✓ when RRULE | iCalendar RRULE syntax |
| `schedules[].calendarId` | UUID | ✓ when CALENDAR | FK → `calendars.id` |
| `schedules[].allDay` | boolean | — | default `true` |
| `schedules[].timeOfDayStart` | string | ✓ when allDay=false | format `HH:mm` |
| `schedules[].timeOfDayEnd` | string | ✓ when allDay=false | format `HH:mm` |
| `schedules[].timezone` | string | — | IANA tz name; default `Asia/Bangkok` |

---

## Response Body `200 OK`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleId": "DR-20260424-001",
    "status": "ACTIVE",
    "schedules": [
      {
        "scheduleId": "s1000000-0000-0000-0000-000000000001",
        "placementId": "p1000000-0000-0000-0000-000000000001",
        "placementName": "Home Banner",
        "channelId": "ch100000-0000-0000-0000-000000000001",
        "channelName": "Mobile App",
        "startDate": "2026-04-30T17:00:00Z",
        "endDate": "2026-05-31T16:59:59Z",
        "recurrenceType": "ONCE",
        "allDay": true,
        "timezone": "Asia/Bangkok",
        "isActive": true
      },
      {
        "scheduleId": "s1000000-0000-0000-0000-000000000002",
        "placementId": "p1000000-0000-0000-0000-000000000002",
        "placementName": "Loan Banner",
        "channelId": "ch100000-0000-0000-0000-000000000002",
        "channelName": "Web",
        "startDate": "2026-05-31T17:00:00Z",
        "endDate": "2026-06-30T16:59:59Z",
        "recurrenceType": "CALENDAR",
        "calendarId": "ca100000-0000-0000-0000-000000000001",
        "allDay": true,
        "timezone": "Asia/Bangkok",
        "isActive": true
      }
    ]
  }
}
```

---

## Validation Errors

| Case | HTTP | Code | Message |
|------|------|------|---------|
| `id` ไม่มีใน DB | 404 | `NOT_FOUND` | `decision rule not found` |
| `placementId` ไม่มีใน DB | 404 | `NOT_FOUND` | `placement {id} not found` |
| `calendarId` ไม่มีใน DB | 404 | `NOT_FOUND` | `calendar {id} not found` |
| `startDate >= endDate` | 400 | `INVALID_FIELD` | `startDate must be before endDate` |
| `endDate` อยู่ในอดีต | 400 | `INVALID_FIELD` | `endDate must be a future date` |
| `timeOfDayStart` >= `timeOfDayEnd` | 400 | `INVALID_FIELD` | `timeOfDayStart must be before timeOfDayEnd` |
| `allDay=false` แต่ไม่มี `timeOfDayStart`/`timeOfDayEnd` | 400 | `INVALID_FIELD` | `timeOfDayStart and timeOfDayEnd are required when allDay is false` |
| `recurrenceType=RRULE` แต่ไม่มี `recurrenceRule` | 400 | `INVALID_FIELD` | `recurrenceRule is required when recurrenceType is RRULE` |
| `recurrenceType=CALENDAR` แต่ไม่มี `calendarId` | 400 | `INVALID_FIELD` | `calendarId is required when recurrenceType is CALENDAR` |
| placement มี schedule ครบ 3 แล้ว | 422 | `VALIDATION_ERROR` | `placement {id} has reached the maximum of 3 schedules` |
| `(id, placementId)` pair ซ้ำ | 422 | `VALIDATION_ERROR` | `schedule for this decision rule and placement already exists` |
| `placementId` ซ้ำใน request | 422 | `VALIDATION_ERROR` | `duplicate placementId {id} in schedules` |

---

## DB Mapping

| JSON Field | Table | Column | Notes |
|------------|-------|--------|-------|
| `{id}` (path) | `decision_rules` | `ID` | validate exists; UPDATE `STATUS` → `ACTIVE` |
| `schedules[].placementId` | `placements` | `ID` | validate exists |
| `schedules[].calendarId` | `calendars` | `ID` | validate exists when provided |
| `schedules[].startDate` | `schedules` | `EFFECTIVE_FROM` | convert to UTC before store |
| `schedules[].endDate` | `schedules` | `EFFECTIVE_UNTIL` | convert to UTC before store |
| `schedules[].recurrenceType` | `schedules` | `RECURRENCE_TYPE` | ONCE / RRULE / CALENDAR |
| `schedules[].recurrenceRule` | `schedules` | `RECURRENCE_RULE` | iCal RRULE string |
| `schedules[].calendarId` | `schedules` | `CALENDAR_ID` | |
| `schedules[].allDay` | `schedules` | `ALL_DAY` | |
| `schedules[].timeOfDayStart` | `schedules` | `TIME_OF_DAY_START` | |
| `schedules[].timeOfDayEnd` | `schedules` | `TIME_OF_DAY_END` | |
| `schedules[].timezone` | `schedules` | `TIMEZONE` | default `Asia/Bangkok` |
| — | `schedules` | `IS_ACTIVE` | hardcoded `true` |

### Capacity Check (per placement)

```sql
SELECT COUNT(*) FROM schedules
WHERE placement_id = :placementId
  AND deleted_at IS NULL;
-- reject if count >= 3
```
