# GET /decision-rules

ดึงรายการ `DecisionRule` พร้อม pagination และ filtering

---

## Query Parameters

| Parameter | Type | Required | Valid Values | Notes |
|-----------|------|----------|-------------|-------|
| `type` | string | — | `MASS`, `AUDIENCE`, `SALES_TARGET`, `NON_SALES` | filter by `decision_rules.type` |
| `evaluateType` | string | — | `SCORING`, `SEGMENT`, `ELIGIBLE` | filter by `decision_rules.evaluate_type` |
| `status` | string | — | `DRAFT`, `ACTIVE`, `INACTIVE` | filter by `decision_rules.status` |
| `keyword` | string | — | — | partial match บน `name` และ `decision_rule_id` |
| `page` | int | — | ≥ 1 | default 1 |
| `limit` | int | — | 1–100 | default all data |

**ตัวอย่าง:**
```
GET /decision-rules?type=AUDIENCE&status=ACTIVE&keyword=gold&page=1&limit=20
```

---

## Response Body `200 OK`

```json
{
  "code": "SUCCESS",
  "data": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "decisionRuleId": "DR-20260424-001",
      "name": "Gold Customer Rule",
      "type": "AUDIENCE",
      "evaluateType": "SCORING",
      "campaignCode": "CAMP2026Q2",
      "status": "ACTIVE",
      "subStatus": "N/A",
      "placements": [
        {
          "placementId": "p1000000-0000-0000-0000-000000000001",
          "placementName": "Home Banner",
          "channelName": "Mobile App"
        }
      ],
      "createdBy": { "userId": "u1000000-0000-0000-0000-000000000001", "nameTh": "สมชาย ใจดี", "nameEn": "Somchai Jaidee" },
      "updatedBy": { "userId": "u1000000-0000-0000-0000-000000000002", "nameTh": "สมหญิง รักดี", "nameEn": "Somying Rakdee" },
      "inactiveBy": null,
      "createdAt": "2026-04-24T10:00:00+07:00",
      "updatedAt": "2026-04-24T10:05:00+07:00"
    },
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "decisionRuleId": "DR-20260423-001",
      "name": "Standard Customer Rule",
      "type": "MASS",
      "evaluateType": "ELIGIBLE",
      "campaignCode": "",
      "status": "DRAFT",
      "subStatus": "Missing attribute registry",
      "placements": [],
      "createdBy": { "userId": "u1000000-0000-0000-0000-000000000001", "nameTh": "สมชาย ใจดี", "nameEn": "Somchai Jaidee" },
      "updatedBy": { "userId": "u1000000-0000-0000-0000-000000000001", "nameTh": "สมชาย ใจดี", "nameEn": "Somchai Jaidee" },
      "inactiveBy": null,
      "createdAt": "2026-04-23T09:00:00+07:00",
      "updatedAt": "2026-04-23T09:00:00+07:00"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "totalItems": 2,
    "totalPages": 1
  }
}
```

### Response Field Reference

| Field | Source | Notes |
|-------|--------|-------|
| `id` | `decision_rules.id` | UUID primary key |
| `decisionRuleId` | `decision_rules.decision_rule_id` | business key string |
| `type` | `decision_rules.type` | DecisionType enum |
| `evaluateType` | `decision_rules.evaluate_type` | EvaluateType enum |
| `status` | `decision_rules.status` | DRAFT / ACTIVE / INACTIVE |
| `subStatus` | `decision_rules.sub_status` | N/A / Missing attribute registry |
| `placements` | `schedules` JOIN `placements` JOIN `channels` | aggregated; empty array เมื่อไม่มี schedule |
| `placements[].channelName` | `channels.channel_name` | |
| `createdBy` | `decision_rules.created_by` → `users` | `null` ถ้าไม่มีข้อมูล |
| `createdBy.userId` | `users.id` | UUID ของผู้สร้าง |
| `createdBy.nameTh` | `users.name_th` | |
| `createdBy.nameEn` | `users.name_en` | |
| `updatedBy` | `decision_rules.updated_by` → `users` | `null` ถ้าไม่มีข้อมูล |
| `updatedBy.userId` | `users.id` | UUID ของผู้แก้ไขล่าสุด |
| `updatedBy.nameTh` | `users.name_th` | |
| `updatedBy.nameEn` | `users.name_en` | |
| `inactiveBy` | `decision_rules.inactive_by` → `users` | `null` เมื่อยังไม่ถูก deactivate |
| `inactiveBy.userId` | `users.id` | UUID ของผู้ deactivate |
| `inactiveBy.nameTh` | `users.name_th` | |
| `inactiveBy.nameEn` | `users.name_en` | |

---

## Query Strategy

ใช้ **3 queries แยกกัน** — ไม่มี JOIN users ใน main query และไม่มี N+1

```sql
-- 1. Main query (paginated)
SELECT
  dr.id,
  dr.decision_rule_id,
  dr.name,
  dr.type,
  dr.evaluate_type,
  dr.campaign_code,
  dr.status,
  dr.sub_status,
  dr.created_by,
  dr.updated_by,
  dr.inactive_by,
  dr.created_at,
  dr.updated_at
FROM decision_rules dr
WHERE dr.deleted_at IS NULL
  AND (:type     = '' OR dr.type          = :type)
  AND (:evalType = '' OR dr.evaluate_type = :evalType)
  AND (:status   = '' OR dr.status        = :status)
  AND (:keyword  = '' OR dr.name          ILIKE '%' || :keyword || '%'
                      OR dr.decision_rule_id ILIKE '%' || :keyword || '%')
ORDER BY dr.created_at DESC
LIMIT :limit OFFSET :offset;

-- 2. Placements per decision rule (batch, ไม่ใช้ N+1)
SELECT
  s.decision_rule_id,
  p.id    AS placement_id,
  p.placement_name,
  c.channel_name
FROM schedules s
JOIN placements p ON s.placement_id = p.id AND p.deleted_at IS NULL
JOIN channels  c ON p.channel_id    = c.id AND c.deleted_at IS NULL
WHERE s.decision_rule_id IN (:drIds)
  AND s.deleted_at IS NULL;

-- 3. User name lookup (batch, ไม่ใช้ N+1)
--    collect unique UUIDs จาก created_by + updated_by + inactive_by ทุก row แล้ว query ครั้งเดียว
SELECT id, name_th, name_en
FROM users
WHERE id IN (:userIds)
  AND deleted_at IS NULL;
```

**หมายเหตุ:**
- User names **ไม่ได้ JOIN** ใน main query และ **ไม่ได้ GORM preload** บน entity — ใช้ `UserRepository.FindByIDs` batch แยกต่างหาก
- Application layer collect UUID ที่ไม่ซ้ำจากทั้ง 3 audit field ก่อน แล้วค่อย query users 1 ครั้ง
- Application layer group placement rows ตาม `decision_rule_id`
- ทั้ง placements และ users ใช้ pattern **WHERE IN batch → map ใน Go** — ไม่มี N+1
