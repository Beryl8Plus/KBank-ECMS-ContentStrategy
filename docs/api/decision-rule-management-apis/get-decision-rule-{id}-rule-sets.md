# GET /decision-rules/{id}/rule-sets

ดึง column definitions (จาก Step 1 conditions) และ rule sets ที่มีอยู่ (Wizard Step 2 — read)

---

## Constraints

- `columns` มาจาก `rule_conditions` ที่เป็น template (`rule_id IS NULL`) เรียงตาม `sequence`
- `columns` แสดงเฉพาะ `type: "condition"` rows — ไม่รวม group container rows
- `ruleSets` มาจาก `rules` join `rule_attributes` join `attributes` ของ decision rule นี้
- `values` ใน ruleSet แต่ละ row จะมีจำนวนเท่ากับ `columns` เสมอ (null เมื่อยังไม่ได้กรอก)

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
    "columns": [
      {
        "conditionId": "c1000000-0000-0000-0000-000000000001",
        "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3302",
        "attributeName": "Customer Segment",
        "attributeIsActive": true,
        "logicalOperator": "IN",
        "dataType": "Text"
      },
      {
        "conditionId": "c1000000-0000-0000-0000-000000000002",
        "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3303",
        "attributeName": "Account Balance",
        "attributeIsActive": false,
        "logicalOperator": ">",
        "dataType": "Number"
      },
      {
        "conditionId": "c1000000-0000-0000-0000-000000000004",
        "attributeId": "3f2504e0-4f89-11d3-9a0c-0305e82c3305",
        "attributeName": "Transaction Count",
        "attributeIsActive": true,
        "logicalOperator": ">=",
        "dataType": "Number"
      }
    ],
    "ruleSets": [
      {
        "ruleId": "r1000000-0000-0000-0000-000000000001",
        "orderNo": 1,
        "score": 80,
        "variationName": "HNW Wealth",
        "values": [
          { "conditionId": "c1000000-0000-0000-0000-000000000001", "value": "Gold" },
          { "conditionId": "c1000000-0000-0000-0000-000000000002", "value": "500000" },
          { "conditionId": "c1000000-0000-0000-0000-000000000004", "value": null }
        ]
      },
      {
        "ruleId": "r1000000-0000-0000-0000-000000000002",
        "orderNo": 2,
        "score": 60,
        "variationName": "HNW Non-Wealth",
        "values": [
          { "conditionId": "c1000000-0000-0000-0000-000000000001", "value": "Silver" },
          { "conditionId": "c1000000-0000-0000-0000-000000000002", "value": null },
          { "conditionId": "c1000000-0000-0000-0000-000000000004", "value": "10" }
        ]
      }
    ]
  }
}
```

### Response Field Reference

| Field | Source | Notes |
|-------|--------|-------|
| `columns[].conditionId` | `rule_conditions.id` | ใช้เป็น key ใน PUT step 2 |
| `columns[].attributeName` | `attributes.display_name` | JOIN via `attribute_id` |
| `columns[].attributeIsActive` | `attributes.is_active` | false = warn user |
| `columns[].dataType` | `attributes.data_type` | Text / Number / Date / Boolean |
| `ruleSets[].ruleId` | `rules.id` | UUID; ส่งกลับใน PUT step 2 เพื่อ update |
| `ruleSets[].variationName` | `rules.variation_name` | |
| `ruleSets[].values[].conditionId` | `rule_conditions.id` | อ้างอิง column เดียวกัน |
| `ruleSets[].values[].value` | `rule_attributes.value` | null เมื่อยังไม่กรอก |

---

## Validation Errors

| Case | HTTP | Message |
|------|------|---------|
| `id` ไม่มีใน DB | 404 | `decision rule not found` |
| ยังไม่ผ่าน Step 1 (ไม่มี conditions) | 422 | `decision rule has no conditions defined` |

---

## Query Strategy

```sql
-- 1. Columns: template conditions (type=condition only, เรียง sequence)
SELECT
  rc.id           AS condition_id,
  rc.attribute_id,
  rc.logical_operator,
  a.display_name  AS attribute_name,
  a.data_type,
  a.is_active     AS attribute_is_active
FROM rule_conditions rc
JOIN attributes a ON rc.attribute_id = a.id AND a.deleted_at IS NULL
WHERE rc.decision_rule_id = :id
  AND rc.rule_id IS NULL
  AND rc.attribute_id IS NOT NULL
  AND rc.deleted_at IS NULL
ORDER BY rc.sequence;

-- 2. Rule sets
SELECT id AS rule_id, variation_name, score, order_no
FROM rules
WHERE decision_rule_id = :id AND deleted_at IS NULL
ORDER BY order_no;

-- 3. Values per rule (single query, join ใน application layer)
SELECT
  ra.rule_id,
  ra.rule_condition_id  AS condition_id,
  ra.value
FROM rule_attributes ra
WHERE ra.rule_id IN (:rule_ids)
  AND ra.deleted_at IS NULL;
```

> **หมายเหตุ:** `rule_attributes` ต้องมี column `RULE_CONDITION_ID` (FK → `rule_conditions.id`) เพื่อ map value กลับไปยัง column ได้ถูกต้อง
