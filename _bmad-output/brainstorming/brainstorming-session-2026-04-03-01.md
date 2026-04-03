---
stepsCompleted: [1, 2, 3]
inputDocuments: ["src/diagram/models.md", "src/api/save-decision-rule.md", "src/api/decision-rule-management-apis/save-decision-rule-step-2.md", "docs/api/decision-rule-management-apis/save-decision-rule-step-3.md"]
session_topic: "Design decision rule APIs — step 1 (conditions), step 2 (rule-sets with attribute values), and step 3 (scheduling)"
session_goals: "Define clean, flexible JSON structures for all three save steps that map to decision_rules, rules, rule_condition_groups, rule_condition, and schedules tables"
selected_approach: "ai-recommended"
techniques_used: ["First Principles Thinking", "Morphological Analysis", "Decision Tree Mapping"]
ideas_generated: ["Idea #1: Operator at condition level", "Idea #2: Backend normalize to nested groups", "Idea #3: Hybrid dual storage", "Idea #4: Connector at condition level + group as visual container (SELECTED)", "Idea #5: Raw JSON payload + normalized relational", "Step2 Idea #1: GET flat columns + values array (SELECTED)", "Step2 Idea #2: GET key-value map per rule", "Step2 Idea #3: GET nested with condition group context", "Step2 Idea #4: POST clean save with conditionId references (SELECTED)", "Step3 Idea #1: schedules array per request with max 3 per placement (SELECTED)"]
context_file: "src/diagram/models.md"
---

# Brainstorming Session Results

**Facilitator:** Nrtdemo
**Date:** 2026-04-03

## Session Overview

**Topic:** Design POST /decision-rule/create API request body — how to structure rule conditions for saving into database
**Goals:** Define a clean, flexible JSON request body structure that maps to decision_rules, rules, rule_condition_groups, and rule_condition tables

### Context Guidance

- Database schema from `src/diagram/models.md` with 4 key tables: `decision_rules`, `rules`, `rule_condition_groups`, `rule_condition`
- UI reference image showing Attribute Group with mixed AND/OR operators between conditions and nested groups

### Session Setup

- **Atomic save:** All tables saved in 1 request
- **Structure:** Nested tree (not flat)

---

## Phase 1: First Principles Thinking

### Key Findings

1. `rule_condition_groups` has self-referencing `parent_rule_condition_groups` → **true tree structure**
2. `rule_condition` links `decision_rule_id`, `rule_id`, `rule_condition_group_id`, and `attribute_id` → condition knows all its parents
3. `rules.decision_rule_id` was commented out → needed to uncomment (relationship via `rule_condition` was indirect)
4. `rules` has `variation_name`, `score`, `order_no` → rules are "variations" of a decision rule with ordering

### Critical Discovery: Model Mismatch

From UI analysis: **within 1 group, AND and OR operators can be mixed between conditions**.

But `rule_condition_groups.rule_operator` stores **only one value** per group (AND or OR).

**This means the original model cannot represent the UI behavior directly.**

---

## Phase 2: Morphological Analysis — Ideas Generated

### Idea #1: Move operator to condition level
- Add `logical_connector` to `rule_condition` — operator connects this condition to the **next** item
- Last condition in group has no connector
- **Pro:** Simple, direct mapping
- **Con:** Requires model change

### Idea #2: Backend normalize to nested groups automatically
- Frontend sends mixed operators → Backend transforms into pure AND/OR groups
- e.g., `A OR B AND (Nested)` → `(A OR B) AND (Nested)` = 2 nested groups
- **Pro:** Model stays the same
- **Con:** Render back to UI ≠ original layout (REJECTED — user requires UI round-trip fidelity)

### Idea #3: Hybrid — store both raw + normalized
- Store raw JSON for UI rendering + normalized relational for evaluation engine
- **Con:** Dual storage, sync complexity

### Idea #4: Connector at condition level + group as visual container (SELECTED)
- `rule_condition_groups` = **visual container only** (maps to "Attribute Group" and "Nested Group" in UI)
- `rule_condition_groups.rule_operator` → removed/optional
- Each `rule_condition` has `connector_operator` (AND/OR/null) connecting it to the next item in sequence
- `rule_condition.sequence` determines order within group
- **Storage = UI structure** → render back 100% identical
- Evaluation engine reads conditions + connectors to build expression tree

### Idea #5: Store raw JSON payload + normalized relational
- Add `decision_rules.raw_ui_payload jsonb` for UI rendering
- Normalize into relational tables for evaluation
- **Rejected:** Unnecessarily complex vs Idea #4

---

## Phase 3: Decision Tree Mapping — Final Dimensions

### Dimension Decisions

| Dimension | Choice | Rationale |
|---|---|---|
| **ID Strategy** | No client IDs, server generates | Simpler frontend, no temp ID management |
| **Nesting Style** | Full nested JSON tree | Matches recursive UI component structure |
| **Condition-to-Group** | Conditions inside group object (children) | Natural tree representation |
| **Nested Group Ref** | Nested group as item in conditions array with `type: "nestedGroup"` | Uniform sequence handling |
| **Connector Position** | On each condition, connects to next item | Matches UI toggle between condition rows |
| **Value Format** | F1: Raw value, backend infers from operator | Simplest for frontend |
| **Max Depth** | 3 levels | Business constraint |
| **Max Conditions/Group** | 10 items | Business constraint |
| **Rules** | 1 decision rule → multiple rule variations | Confirmed from model |

---

## Final Output — Step 1

Updated files:
- `src/api/save-decision-rule.md` — Complete request body with JSON example, constraints, DB mapping table, and model changes
- `src/diagram/models.md` — Applied 3 model changes:
  1. `rules.decision_rule_id` — added FK to `decision_rules.id`
  2. `rule_condition.connector_operator` — added enum (AND, OR, nullable)
  3. `rule_condition_groups.rule_operator` — removed (group is visual container only)

---

## Step 2 API Design — Rule Sets with Attribute Values

### Context

- After step 1 saves `decision_rules` + `rule_condition_groups` + condition structure (attribute columns + operators), step 2 handles the **rule rows** (each row = 1 rule variation with attribute values filled in).
- UI shows a table where:
  - **Columns** = attributes from step 1 conditions (e.g., "Attribute 1, Sandbox / Equals (=)")
  - **Rows** = rule variations (Rule No. 1, 2, 3...) each with Score + Variation name
  - **Cells** = attribute values per rule row

### Data Flow Understanding

| Layer | DB Table | UI Representation |
|-------|----------|-------------------|
| **1st Layer — rules** | `rules` (id, decision_rule_id, variation_name, score, order_no) | Each **row** in the table |
| **2nd Layer — attributes** | `rule_condition` (id, rule_id, attribute_id, value...) | Each **cell** (attribute value column) in that row |

### Ideas Generated

#### GET /decision-rule/{id}/rule-sets

**Idea #1: Flat columns + values array (SELECTED)**
- `columns[]` — attribute definitions from step 1 (conditionId, attributeName, logicalOperator, dataType)
- `ruleSets[]` — existing rule rows with `values[]` array mapped by conditionId to columns
- Frontend renders column headers from `columns`, rows from `ruleSets`, cells from `values` by matching conditionId
- **Pro:** Clear separation of column definitions and row data, array preserves order

**Idea #2: Key-value map per rule**
- `values` as `{ "condition-uuid": "value" }` map instead of array
- **Pro:** Easier frontend access by key
- **Con:** Loses explicit ordering

**Idea #3: Nested with condition group context**
- Keeps `conditionGroups` structure in response
- **Rejected:** Over-complex for table view, step 2 doesn't need group hierarchy

#### POST /decision-rule/save/step-2

**Idea #4: Clean save with conditionId references (SELECTED)**
- `ruleId: null` → create new rule row; non-null → update existing
- `conditions[]` uses `conditionId` referencing `rule_condition.id` from step 1
- Backend sets `rule_condition.rule_id` to link condition values to their rule row
- Frontend knows existing vs new rows from GET response (`ruleId` presence)

### Decisions

| Decision | Choice | Rationale |
|---|---|---|
| **GET response structure** | Flat columns + values array (Idea #1) | Clear column/row separation, array preserves order |
| **Include conditionGroupId** | No | Step 2 only cares about attribute columns, not group hierarchy |
| **POST ruleId null handling** | null = create, non-null = update | Frontend discovers existing ruleIds from GET response |
| **Value reference key** | conditionId | Direct FK to rule_condition.id from step 1 |

### Final Output — Step 2

Updated file:
- `src/api/decision-rule-management-apis/save-decision-rule-step-2.md` — Complete GET response + POST request body with JSON examples, constraints, and DB mapping tables

---

## Step 3 API Design — Scheduling

### Context

- After step 2 saves rule rows with attribute values, step 3 handles **scheduling** — assigning a decision rule to one or more placements with a time window.
- UI sends `decisionRuleId` + an array of `schedules`, each with `placementId`, `startDate`, `endDate`.
- **Business constraint:** Maximum 3 schedules per placement (across all decision rules).

### Data Flow Understanding

| Layer | DB Table | UI Representation |
|-------|----------|-------------------|
| **Decision Rule** | `decision_rules` | Status updated to `ACTIVE` on save |
| **Schedule rows** | `schedules` | Each placement time window |
| **Placement** | `placements` | Master data — name/description |

### Idea #1: Schedules array per request (SELECTED)

- Top-level `decisionRuleId` + `schedules[]` array, each item = `{ placementId, startDate, endDate }`
- Capacity check: `SELECT COUNT(*) FROM schedules WHERE placement_id = :placementId` before each INSERT; reject 422 if >= 3
- Unique constraint on `(decision_rule_id, placement_id)` — both DB-level and API validation
- `startDate` < `endDate` validated server-side
- On success, `decision_rules.status` set to `ACTIVE`
- Response echoes back `scheduleId` and `isActive: true` for each created record

### Decisions

| Decision | Choice | Rationale |
|---|---|---|
| **Request shape** | `schedules[]` array under `decisionRuleId` | One atomic call creates all schedule rows |
| **Capacity scope** | Per `placementId` across all decision rules | Prevents placement overload globally |
| **Duplicate handling** | 422 with clear message | `(decisionRuleId, placementId)` unique index |
| **Status transition** | `ACTIVE` on step-3 save | Step 3 is the activation gate |
| **Timestamp format** | `timestamptz` with timezone offset | Matches `schedules.start_timestamp` / `end_timestamp` column type |

### Final Output — Step 3

Created file:
- `docs/api/decision-rule-management-apis/save-decision-rule-step-3.md` — Request body, response body, validation error table, and DB mapping including capacity check SQL
