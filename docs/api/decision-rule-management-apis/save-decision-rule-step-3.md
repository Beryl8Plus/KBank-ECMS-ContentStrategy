# PUT /decision-rules/{id}/schedules

กำหนด `Schedule` และ `Placement` สำหรับ Decision Rule (Wizard Step 3)

Endpoint นี้ใช้ได้ทั้ง **Create Mode** และ **Edit Mode** — ทำ **full replacement** เสมอ

---

## Constraints

- **Full Replacement:** backend จะ **DELETE** schedules ทั้งหมดของ DR นี้ก่อน แล้ว INSERT ใหม่จาก request
- `scheduleId` ใน request เป็น optional — frontend ใช้ track ได้ แต่ backend ไม่ใช้ (generate ID ใหม่ทุกครั้ง)
- Max **3 schedules ต่อ placement** (นับเฉพาะ DR **อื่น** ที่ใช้ placement เดียวกัน หลังจากของ DR นี้ถูกลบแล้ว)
- `startDate` ต้องน้อยกว่า `endDate`
- `endDate` ต้องไม่อยู่ในอดีต
- `placementId` ต้องไม่ซ้ำภายใน request เดียวกัน
- `timezone` default เป็น `Asia/Bangkok` หาก frontend ไม่ส่งมา
- `timeOfDayStart` / `timeOfDayEnd` format: `HHmm` (24-hour) เช่น `"0800"`, `"2359"`
- `allDay: true` → ไม่ต้องส่ง `timeOfDayStart` / `timeOfDayEnd`
- `recurrenceType: "RRULE"` → ต้องส่ง `recurrenceRule` (iCalendar RRULE format)
- `recurrenceType: "CALENDAR"` → ต้องส่ง `calendarId`
- `recurrenceType: "ONCE"` → ทำงานครั้งเดียวตาม `startDate`–`endDate`
- Request นี้ **ไม่เปลี่ยน** `decision_rules.status` — ใช้ Step 4 (`PUT /decision-rules/{id}/activate`) เพื่อ Activate
- ทุก operation ทำภายใน **single database transaction**

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
      "scheduleId": null,
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

**Advanced (RRULE, Time window) — Edit Mode ส่ง scheduleId ที่มีอยู่แล้ว:**
```json
{
  "schedules": [
    {
      "scheduleId": "s1000000-0000-0000-0000-000000000001",
      "placementId": "p1000000-0000-0000-0000-000000000001",
      "startDate": "2026-05-01T00:00:00+07:00",
      "endDate": "2026-05-31T23:59:59+07:00",
      "recurrenceType": "RRULE",
      "recurrenceRule": "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
      "allDay": false,
      "timeOfDayStart": "0800",
      "timeOfDayEnd": "1800",
      "timezone": "Asia/Bangkok"
    },
    {
      "scheduleId": null,
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
| `schedules[].scheduleId` | UUID \| null | — | frontend tracking only; backend ไม่ใช้ (full replacement ทุกครั้ง) |
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
    "status": "DRAFT",
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

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| `placementId` ไม่มีใน DB | 404 | `placement {id} not found` |
| `calendarId` ไม่มีใน DB | 404 | `calendar {id} not found` |
| `startDate >= endDate` | 422 | `startDate must be before endDate for placement {id}` |
| `endDate` อยู่ในอดีต | 422 | `endDate must be a future date for placement {id}` |
| `allDay=false` แต่ไม่มี `timeOfDayStart`/`timeOfDayEnd` | 422 | `timeOfDayStart and timeOfDayEnd required when allDay is false for placement {id}` |
| `recurrenceType=RRULE` แต่ไม่มี `recurrenceRule` | 422 | `recurrenceRule is required when recurrenceType is RRULE` |
| `recurrenceType=CALENDAR` แต่ไม่มี `calendarId` | 422 | `calendarId is required when recurrenceType is CALENDAR` |
| placement (ของ DR อื่น) มี schedule ครบ 3 แล้ว | 422 | `placement {id} has reached the maximum of 3 schedules` |
| `placementId` ซ้ำใน request | 422 | `duplicate placementId {id} in schedules` |

---

## DB Mapping

| JSON Field | Table | Column | Notes |
|------------|-------|--------|-------|
| `{id}` (path) | `decision_rules` | `ID` | validate exists; **ไม่เปลี่ยน STATUS** |
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

เนื่องจาก SaveStep3 ทำ full replacement (ลบ schedules ของ DR นี้ก่อน) การนับ cap จึงนับเฉพาะ DR อื่น:

```sql
SELECT COUNT(*) FROM schedules
WHERE placement_id = :placementId
  AND decision_rule_id != :currentDRId
  AND deleted_at IS NULL;
-- reject if count >= 3
```

### Full Replacement Logic

```
BEGIN TRANSACTION
  DELETE schedules WHERE DECISION_RULE_ID = {id}
  INSERT INTO schedules ... (schedules from request, new IDs generated)
COMMIT
```
