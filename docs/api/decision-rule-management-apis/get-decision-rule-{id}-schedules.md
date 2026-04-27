# GET /decision-rules/{id}/schedules

ดึงข้อมูล schedules ทั้งหมดของ `DecisionRule` ที่ระบุ (Wizard Step 3)

---

## Constraints

- คืนข้อมูล `schedules` ที่ยังไม่ถูก soft delete (`deleted_at IS NULL`)
- `schedules` จะว่างหาก step 3 ยังไม่ถูก save
- Query ใช้ Preload แบบ batch เพื่อหลีกเลี่ยง N+1

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
      }
    ],
    "createdAt": "2026-04-24T10:00:00+07:00",
    "updatedAt": "2026-04-24T10:05:00+07:00"
  }
}
```

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |

---

## Query Strategy (หลีกเลี่ยง N+1)

```go
// ใน Repository layer — Preload schedules พร้อม placement และ channel ใน round trips เท่าที่จำเป็น
db.WithContext(ctx).
    Preload("Schedules", func(db *gorm.DB) *gorm.DB {
        return db.Joins("Placement").
            Joins("Placement.Channel").
            Where("deleted_at IS NULL")
    }).
    First(&decisionRule, "id = ? AND deleted_at IS NULL", id)
```

**Round trips ที่เกิดขึ้น:**
1. `decision_rules` WHERE id = ?
2. `schedules` WHERE decision_rule_id = ? (+ JOIN placements + channels)

รวม **2 queries** — ไม่มี N+1
