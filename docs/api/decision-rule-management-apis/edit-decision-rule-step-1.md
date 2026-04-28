# PUT /decision-rules/{id}

แก้ไข header และ condition tree ของ `DecisionRule` ที่มีอยู่ (Wizard Step 1 — Edit Mode)

---

## Constraints

- ใช้กับ `DecisionRule` ที่มี `status = DRAFT` เท่านั้น
- Request ส่ง **complete desired state** ของ conditions ทั้งหมด — backend คำนวณ diff เอง
  - `conditionId` มีค่า → **UPDATE** condition นั้น
  - `conditionId: null` → **INSERT** condition ใหม่
  - `conditionId` ที่อยู่ใน DB แต่ไม่อยู่ใน request → **DELETE** + **Cascade Delete** Step 2
- **Cascade Delete Rule:** เมื่อ condition ถูกลบ ระบบจะลบ `Rule` (และ `RuleAttribute`) ทั้งหมดใน Step 2 ที่มี `rule_attributes.attribute_id` ตรงกับ `attribute_id` ของ condition ที่ถูกลบ
- **Option A — No Duplicate Attributes:** แต่ละ `attributeId` ใช้ได้เพียงครั้งเดียวในทุก leaf conditions ของ Decision Rule เดียวกัน ห้ามซ้ำ
- **Active Attributes Only:** `attributeId` ต้องอ้างอิง Attribute ที่มี `is_active = true` — Attribute ที่ถูก deactivate โดย Batch Sync ไม่สามารถถูกเพิ่มหรือแก้ไขใน Rule ได้
- Max nesting depth: **3 ระดับ**
- Max conditions ต่อ group: **10 items**
- `type: "group"` ต้องมี `conditions` array ที่ไม่ว่าง
- `type: "condition"` ต้องมี `attributeId`, `logicalOperator`
- `connectorOperator` ของ item **สุดท้าย** ใน array ต้องเป็น `null`
- `campaignCode` จำเป็นเฉพาะเมื่อ `type` เป็น `AUDIENCE` หรือ `SALES_TARGET`
- ทุก operation ทำภายใน **single database transaction**

---

## Path Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `id` | UUID | ✓ | `decision_rules.id` (primary key) |

---

## Request Body

```json
{
  "type": "AUDIENCE",
  "evaluateType": "SCORING",
  "name": "Gold Customer Rule v2",
  "contentPath": "/content/promo-banner-v2",
  "campaignCode": "CAMP2026Q2",
  "score": 85,
  "conditions": [
    {
      "conditionId": "c1000000-0000-0000-0000-000000000001",
      "type": "condition",
      "sequence": 1,
      "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3302",
      "logicalOperator": ">=",
      "connectorOperator": "AND"
    },
    {
      "conditionId": null,
      "type": "condition",
      "sequence": 2,
      "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3309",
      "logicalOperator": "IN",
      "connectorOperator": null
    }
  ]
}
```

> **หมายเหตุ:** `conditionId` ของ condition เดิม `"c1000000-0000-0000-0000-000000000002"` ไม่อยู่ใน request
> → ระบบจะ DELETE condition นั้น และ Cascade Delete Rules ใน Step 2 ที่ใช้ attribute เดียวกัน

### Field Reference

| Field | Type | Required | Valid Values | Notes |
|-------|------|----------|-------------|-------|
| `type` | string | ✓ | `MASS`, `AUDIENCE`, `SALES_TARGET`, `NON_SALES` | |
| `evaluateType` | string | ✓ | `SCORING`, `SEGMENT`, `ELIGIBLE` | |
| `name` | string | ✓ | max 255 chars | |
| `contentPath` | string | ✓ | max 255 chars | |
| `campaignCode` | string | ✓ when IsCampaign | max 25 chars | required for AUDIENCE, SALES_TARGET |
| `score` | float | — | 0–100 | default 0 |
| `conditions[].conditionId` | UUID \| null | — | — | null = insert new; non-null = update existing |
| `conditions[].type` | string | ✓ | `condition`, `group` | |
| `conditions[].sequence` | int | ✓ | ≥ 1 | ordering within parent |
| `conditions[].attributeId` | UUID | ✓ for condition | — | must be unique across all conditions in this DR |
| `conditions[].logicalOperator` | string | ✓ for condition | `<`, `<=`, `>`, `>=`, `=`, `!=`, `IN`, `BETWEEN` | null for group |
| `conditions[].connectorOperator` | string | — | `AND`, `OR`, `null` | null on last item in array |
| `conditions[].conditions` | array | ✓ for group | — | recursive |

---

## Response Body `200 OK`

### ไม่มี Cascade Effect (ไม่มี condition ถูกลบ)

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleId": "RS-202604-0001",
    "status": "DRAFT",
    "cascadeEffect": null,
    "updatedAt": "2026-04-27T10:00:00+07:00"
  }
}
```

### มี Cascade Effect (มี condition ถูกลบ → Rules ใน Step 2 ถูก Cascade Delete)

```json
{
  "code": "SUCCESS",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "decisionRuleId": "RS-202604-0001",
    "status": "DRAFT",
    "cascadeEffect": {
      "deletedRulesCount": 2,
      "affectedRuleIds": [
        "r1000000-0000-0000-0000-000000000001",
        "r1000000-0000-0000-0000-000000000002"
      ]
    },
    "updatedAt": "2026-04-27T10:00:00+07:00"
  }
}
```

> Frontend ใช้ `cascadeEffect != null && deletedRulesCount > 0` เป็น trigger สำหรับแสดง Warning Alert
> ก่อนที่ user จะ navigate ออกจาก Step 1

### Response Field Reference

| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | `decision_rules.id` |
| `decisionRuleId` | string | business running ID |
| `status` | string | `DRAFT` — ไม่เปลี่ยน status จาก Step 1 edit |
| `cascadeEffect` | object \| null | null ถ้าไม่มี condition ถูกลบ |
| `cascadeEffect.deletedRulesCount` | int | จำนวน Rules ที่ถูกลบจาก Step 2 |
| `cascadeEffect.affectedRuleIds` | UUID[] | IDs ของ Rules ที่ถูกลบ (ใช้แสดง Warning) |
| `updatedAt` | RFC3339 | เวลาที่ update สำเร็จ |

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| `type` ไม่ถูกต้อง | 422 | `type must be one of MASS, AUDIENCE, SALES_TARGET, NON_SALES` |
| `evaluateType` ไม่ถูกต้อง | 422 | `evaluateType must be one of SCORING, SEGMENT, ELIGIBLE` |
| `campaignCode` หายไปเมื่อ IsCampaign | 422 | `campaignCode is required for AUDIENCE and SALES_TARGET types` |
| nesting depth > 3 | 422 | `conditions exceed maximum nesting depth of 3` |
| conditions ต่อ group > 10 | 422 | `group exceeds maximum of 10 conditions` |
| `type=condition` แต่ `attributeId` เป็น null | 422 | `attributeId is required for condition type` |
| `type=group` แต่ `conditions` ว่าง | 422 | `group must contain at least one condition` |
| `connectorOperator` ของ item สุดท้ายไม่เป็น null | 422 | `last condition in array must have connectorOperator null` |
| `attributeId` ซ้ำกันใน conditions | 422 | `attributeId {id} appears more than once in conditions — each attribute may only be used once` |
| `attributeId` ไม่มีใน DB | 404 | `attribute {id} not found` |
| `attributeId` ถูก deactivate | 422 | `attribute {id} ({fieldName}) is inactive and cannot be used in new rules` |

---

## Transaction Flow (within single DB transaction)

```
BEGIN TRANSACTION

1. หา conditions ที่ถูกลบ:
   deleted_condition_ids = existing_ids − incoming_condition_ids

2. ถ้า deleted_condition_ids ไม่ว่าง:
   a. หา attribute_ids ของ conditions ที่ถูกลบ (leaf nodes เท่านั้น)
   b. SELECT DISTINCT r.ID FROM rules r
      INNER JOIN rule_attributes ra ON ra.RULE_ID = r.ID
      WHERE r.DECISION_RULE_ID = {id}
        AND ra.ATTRIBUTE_ID IN (deleted_attribute_ids)
      → affected_rule_ids
   c. DELETE rule_attributes WHERE RULE_ID IN (affected_rule_ids)
   d. DELETE rules WHERE ID IN (affected_rule_ids)
   e. BFS expand descendants ของ deleted_condition_ids (max 3 levels)
   f. DELETE rule_conditions WHERE ID IN (all_descendant_ids)

3. UPSERT conditions (parent nodes ก่อน child nodes เสมอ)

4. UPDATE decision_rules header fields

COMMIT

5. Return affected_rule_ids ใน response
```

---

## DB Mapping

| JSON Field | Table | Column | Operation |
|------------|-------|--------|-----------|
| `{id}` (path) | `decision_rules` | `ID` | validate exists |
| `name` | `decision_rules` | `NAME` | UPDATE |
| `type` | `decision_rules` | `TYPE` | UPDATE |
| `evaluateType` | `decision_rules` | `EVALUATE_TYPE` | UPDATE |
| `contentPath` | `decision_rules` | `CONTENT_PATH` | UPDATE |
| `campaignCode` | `decision_rules` | `CAMPAIGN_CODE` | UPDATE |
| `score` | `decision_rules` | `SCORE` | UPDATE |
| `conditions[n]` (conditionId=null) | `rule_conditions` | — | INSERT |
| `conditions[n]` (conditionId=set) | `rule_conditions` | — | UPDATE WHERE `ID = conditionId` |
| conditions absent from request | `rule_conditions` | `DELETED_AT` | soft-DELETE + cascade |
| cascade affected rules | `rules` | `DELETED_AT` | soft-DELETE |
| cascade affected rule attrs | `rule_attributes` | `DELETED_AT` | soft-DELETE |
