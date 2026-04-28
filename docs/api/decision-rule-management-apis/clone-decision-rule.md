# POST /decision-rules/{id}/clone

ทำสำเนา (Deep Copy) ของ `DecisionRule` ที่มีอยู่ไปเป็น Draft ใหม่

---

## Constraints

- สามารถ Clone ได้จาก `DecisionRule` ที่มี `status` เป็น `DRAFT`, `ACTIVE` หรือ `INACTIVE` ก็ได้ **ยกเว้น** กรณีต่อไปนี้
- **Fail-fast Validation:** หาก `status = INACTIVE` **และ** `subStatus = "Missing attribute registry"` ระบบจะ return error ทันทีโดยไม่อ่านข้อมูลเพิ่มเติม
- ผลลัพธ์ที่ได้จะมี `status = DRAFT` และ `subStatus = "N/A"` เสมอ
- `decisionRuleRunning` (Running Number) จะถูก Generate ใหม่โดยอัตโนมัติผ่าน Postgres function `next_decision_rule_id()`
- ข้อมูลที่ถูก Copy:
  - Decision Rule Header (name, type, evaluateType, contentPath, campaignCode, score)
  - Attribute Conditions (Step 1) — รวมถึง parent-child links ทั้งหมด พร้อม UUID ใหม่
  - Rule Sets & RuleAttributes (Step 2)
  - **Placeholder Schedules (Step 3):** Copy เฉพาะ `placementId` — ข้อมูล schedule (`startDate`, `endDate`, `startTime`, `endTime`) จะถูก **set เป็น null/zero** เพื่อให้ Frontend ใช้ pre-fill ช่อง Placement เมื่อ user ไปถึง Step 3 และให้ user เลือกวันเวลาใหม่เสมอ
- ทุก operation อยู่ใน **single database transaction** (Commit หรือ Rollback ทั้งหมด)
- `InactiveBy` จาก record ต้นฉบับ **ไม่ถูก** Copy

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` ของ record ต้นฉบับ |

---

## Request Body

ไม่มี Request Body

---

## Response Body `201 Created`

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "decisionRuleRunning": "RS-202604-0002",
    "status": "DRAFT",
    "createdAt": "2026-04-27T14:30:00+07:00"
  }
}
```

### Response Field Reference

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | UUID ใหม่ของ DecisionRule ที่ถูก Clone |
| `decisionRuleRunning` | string | Running Number ใหม่ (gen โดย `next_decision_rule_id()`) |
| `status` | string | `DRAFT` เสมอ |
| `createdAt` | RFC3339 | เวลาที่สร้าง record ใหม่ |

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| `status = INACTIVE` และ `subStatus = "Missing attribute registry"` | 422 | `cannot clone a rule with status INACTIVE and sub-status 'Missing attribute registry'` |

---

## Transaction Flow (within single DB transaction)

```
BEGIN TRANSACTION

1. INSERT decision_rules (header ใหม่, status=DRAFT, DecisionRuleRunning=auto-gen)

2. ถ้ามี Conditions:
   - Build old_id → new_id mapping
   - INSERT rule_conditions (UUID ใหม่ทุก record, ParentRuleConditionID ชี้ไปยัง new parent ID)

3. ถ้ามี Rules:
   - Build old_rule_id → new_rule_id mapping
   - INSERT rules (UUID ใหม่, DECISION_RULE_ID = new DR id)

4. ถ้ามี RuleAttributes:
   - INSERT rule_attributes (UUID ใหม่, RULE_ID = new rule id จาก mapping ข้างต้น)

5. ถ้ามี Schedules (Step 3):
   - INSERT schedules (UUID ใหม่, PLACEMENT_ID = เดิม, EFFECTIVE_FROM/UNTIL = zero, TIME_OF_DAY_START/END = null, IS_ACTIVE = false)

COMMIT
```

---

## Deep Copy Scope

| ข้อมูล | Copy | หมายเหตุ |
|--------|------|---------|
| Decision Rule Header | ✓ | name, type, evaluateType, contentPath, campaignCode, score |
| `status` | — | บังคับ `DRAFT` |
| `subStatus` | — | บังคับ `N/A` |
| `decisionRuleRunning` | — | Gen ใหม่อัตโนมัติ |
| `inactiveBy` | — | ไม่ Copy (ตั้งเป็น null) |
| Conditions (Step 1) | ✓ | UUID ใหม่, parent-child links ถูก remap |
| Rules (Step 2) | ✓ | UUID ใหม่ |
| RuleAttributes (Step 2) | ✓ | UUID ใหม่, RuleID ชี้ไปยัง Rule ใหม่ |
| Schedules — PlacementID | ✓ | Carry over เพื่อ pre-fill Step 3 |
| Schedules — EffectiveFrom/Until | — | Set เป็น zero (user ต้องเลือกใหม่) |
| Schedules — TimeOfDayStart/End | — | Set เป็น null |
| Schedules — IsActive | — | บังคับ `false` |

---

## DB Mapping

| Action | Table | Column | Notes |
|--------|-------|--------|-------|
| INSERT | `decision_rules` | `ID` | UUID ใหม่ |
| INSERT | `decision_rules` | `DECISION_RULE_RUNNING` | gen จาก `next_decision_rule_id()` |
| INSERT | `decision_rules` | `STATUS` | hardcoded `DRAFT` |
| INSERT | `decision_rules` | `SUB_STATUS` | hardcoded `N/A` |
| INSERT | `rule_conditions` | `ID` | UUID ใหม่ต่อทุก condition |
| INSERT | `rule_conditions` | `PARENT_RULE_CONDITION_ID` | UUID ที่ remap แล้ว |
| INSERT | `rule_conditions` | `DECISION_RULE_ID` | FK → new DR |
| INSERT | `rules` | `ID` | UUID ใหม่ |
| INSERT | `rules` | `DECISION_RULE_ID` | FK → new DR |
| INSERT | `rule_attributes` | `ID` | UUID ใหม่ |
| INSERT | `rule_attributes` | `RULE_ID` | FK → new Rule (remapped) |
| INSERT | `schedules` | `PLACEMENT_ID` | Copy จาก original |
| INSERT | `schedules` | `EFFECTIVE_FROM` | `0001-01-01 00:00:00+00` (zero time) |
| INSERT | `schedules` | `EFFECTIVE_UNTIL` | `0001-01-01 00:00:00+00` (zero time) |
| INSERT | `schedules` | `TIME_OF_DAY_START` | null |
| INSERT | `schedules` | `TIME_OF_DAY_END` | null |
| INSERT | `schedules` | `IS_ACTIVE` | `false` |
