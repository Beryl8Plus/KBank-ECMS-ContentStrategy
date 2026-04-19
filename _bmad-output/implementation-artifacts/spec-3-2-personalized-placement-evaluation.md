---
title: 'Personalized Placement Evaluation (per-user criteria checking)'
type: 'feature'
created: '2026-04-14'
status: 'done'
baseline_commit: 'feat/microservice-layer HEAD (2026-04-14)'
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The CMS Runtime currently stores pre-scored `ContentResult` objects in Redis (`cms:placement:{name}`) based on rule conditions evaluated against static attribute values. There is no mechanism to check personalization criteria (rule conditions with expected values) against an individual user's attributes at delivery time, and no per-user evaluation cache.

**Approach:** Refactor `evaluatePlacementGroup` to store serializable logic entries (canonical expression + flattened conditions + expected values) instead of `ContentResult`. Add `BuildLogicExpression` (conditions + expected values → canonical string with values). Add a personalized delivery path in `CMSDeliveryService` that receives `cisID`, `userID`, and user attribute values from the request payload, evaluates each logic entry against them, caches per-user boolean results, and returns `ContentResult`.

## Boundaries & Constraints

**Always:**
- Canonical expression format per leaf: `attr_uuid:operator:compact_json_value` (e.g. `(attr_uuid_1:=:"A") AND (attr_uuid_2:>:10)`)
- Logic hash = SHA-256 of the full canonical expression string (with expected values)
- Runtime cache key stays `cms:placement:{name}` → value type changes from `[]ContentResult` to `[]PlacementLogicEntry`
- Personalized placement key (delivery-written): `cms:placement:{cis_id}:{name}`
- Per-user eval cache key: `cms:eval:user:{user_id}:logic:{sha256_of_logic_expr}`
- Per-user eval cache stores `bool` (`"true"` / `"false"` as string)
- User attributes arrive as `map[string]json.RawMessage` (attr UUID → compact JSON value) from request payload
- Existing `GenerateConditionHash` (structure-only hash without values) must remain for backward-compat
- `evaluateConditionGroup` and `compareValues` must not change signature

**Ask First:**
- If `PlacementLogicEntry` struct needs to live in a separate shared package (e.g. `pkg/`) rather than `domain/service`
- If the per-user eval TTL should be configurable (not hardcoded)

**Never:**
- Do not remove `cmsPlacementKey` (used by delivery fallback path)
- Do not change existing `GetContentByPlacements` signature (keep backward compat; new method is additive)
- Do not re-implement a string parser for the logic expression — evaluate from conditions struct, not by parsing the string

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Runtime stores logic entry | `evaluatePlacementGroup` runs, conditions + variation exist | `[]PlacementLogicEntry` JSON written to `cms:placement:{name}` | Skip rule if variation not found |
| Delivery — personalized cache hit | `cms:placement:{cis_id}:{name}` exists in Redis | Unmarshal and return `[]ContentResult` immediately | Unmarshal error → treat as miss |
| Delivery — user eval cache hit | `cms:eval:user:{uid}:logic:{sha256}` = `"true"` | Include entry in results without re-evaluating | — |
| Delivery — user eval cache miss | No user eval cache for this logic hash | Evaluate conditions against `userAttrs`, store `"true"`/`"false"`, include if true | Eval error → skip entry (log warn) |
| Empty logic list | `cms:placement:{name}` exists but empty array | Return empty `[]ContentResult` | — |
| Missing user attr for condition | `userAttrs` does not contain required `attributeID` | Evaluate to `false`; cache as `"false"` | Log at DEBUG |
| No personalized cache — runtime key also miss | Both keys absent | Return empty `[]ContentResult`, no error | — |

</frozen-after-approval>

## Code Map

- `internal/service/evaluator/condition_normalization.go` — add `BuildLogicExpression` and `GenerateLogicHash`
- `internal/service/evaluator/condition_evaluator.go` — add `EvaluateLogicConditions` (user attrs vs stored conditions)
- `internal/domain/service/cms_delivery.go` — add `PlacementLogicEntry`, `LogicCondition` types; add `GetPersonalizedContent` to `DeliveryService` interface
- `internal/service/cms_runtime_service.go` — refactor `evaluatePlacementGroup` to build/store `[]PlacementLogicEntry`; add `cmsPlacementLogicKey`
- `internal/service/cms_delivery_service.go` — add `cmsPersonalizedPlacementKey`, `cmsUserEvalKey`, implement `GetPersonalizedContent`
- `internal/service/evaluator/condition_normalization_test.go` — extend with `BuildLogicExpression` value tests
- `internal/service/cms_delivery_service_test.go` — add personalized flow tests

## Tasks & Acceptance

**Execution:**
- [x] `internal/service/evaluator/condition_normalization.go` -- Add `BuildLogicExpression(conditions []entity.RuleCondition, expectedValues map[string]json.RawMessage) string` (leaf format: `attr_uuid:operator:compact_json_value`); add `GenerateLogicHash` using it -- enables value-aware canonical string and stable hash for user eval cache key
- [x] `internal/domain/service/cms_delivery.go` -- Add `PlacementLogicEntry` and `LogicCondition` structs; add `GetPersonalizedContent(ctx, cisID, userID string, placementNames []string, userAttrs map[string]json.RawMessage) ([]ContentResult, error)` to `DeliveryService` interface -- defines the shared types and contract
- [x] `internal/service/evaluator/condition_evaluator.go` -- Add `EvaluateLogicConditions(conditions []LogicCondition, userAttrs map[string]json.RawMessage) (bool, error)` reusing existing `evalSiblings`/`compareValues` logic but reading actual values from `userAttrs` instead of `c.Attribute.Value` -- enables per-user evaluation without re-fetching DB
- [x] `internal/service/cms_runtime_service.go` -- Rename `cmsPlacementKey` usages in `evaluatePlacementGroup` to `cmsPlacementLogicKey`; refactor to build `[]PlacementLogicEntry` (includes `LogicExpr`, `LogicHash`, flattened `[]LogicCondition` with data types + expected values) instead of `[]ContentResult` -- stores evaluatable logic for delivery layer
- [x] `internal/service/cms_delivery_service.go` -- Add key helpers `cmsPersonalizedPlacementKey(cisID, name)` → `"cms:placement:"+cisID+":"+name` and `cmsUserEvalKey(userID, hash)` → `"cms:eval:user:"+userID+":logic:"+hash`; implement `GetPersonalizedContent`: (1) check `cmsPersonalizedPlacementKey` for cache hit, (2) fetch `cmsPlacementLogicKey`, (3) for each entry check/populate user eval cache, (4) collect passing entries as `ContentResult`, sort desc by score, write to `cmsPersonalizedPlacementKey` with `resultTTL`, return -- implements the full personalized delivery path
- [x] `internal/service/evaluator/condition_normalization_test.go` -- Add tests: `BuildLogicExpression` produces expected format with values; same logic+values = same hash; changing expected value changes hash -- validates value-aware canonical string
- [x] `internal/service/cms_delivery_service_test.go` -- Add tests: personalized cache hit returns result; user eval cache hit skips re-eval; user eval cache miss evaluates and caches; missing attr evaluates to false -- validates I/O edge cases

**Acceptance Criteria:**
- Given a placement group with active schedules, when `evaluatePlacementGroup` runs, then Redis at `cms:placement:{name}` contains a JSON array of `PlacementLogicEntry` with non-empty `logicExpr`, `logicHash`, and `conditions`
- Given a request with `userAttrs` where all conditions pass, when `GetPersonalizedContent` is called, then the result includes the `ContentResult` and `cms:eval:user:{uid}:logic:{hash}` = `"true"`
- Given a subsequent request for the same user+placement, when `GetPersonalizedContent` is called, then `cms:eval:user:{uid}:logic:{hash}` is read from cache (no re-evaluation)
- Given `userAttrs` missing a required attribute UUID, when evaluating, then the entry is excluded from results and `"false"` is cached
- Given `BuildLogicExpression` called twice with identical conditions+values in different slice order, when both results are hashed, then the hashes are equal

## Design Notes

**Canonical string leaf format with value:**
```
Before: attr_uuid_1:=
After:  attr_uuid_1:=:"A"          (text, JSON string)
        attr_uuid_2:>:10            (number, JSON number)
        attr_uuid_3:IN:["v1","v2"]  (array, compact JSON)
```

**`PlacementLogicEntry` (stored at `cms:placement:{name}`):**
```json
{
  "decisionRuleId": "uuid",
  "contentPath": "/banner/hero",
  "score": 9.0,
  "variation": "varA",
  "logicHash": "sha256hex",
  "logicExpr": "((attr1:=:\"A\") AND (attr2:>:10))",
  "source": "DECISION_RULE",
  "startDateTime": "...", "endDateTime": "...",
  "conditions": [
    {"attributeId":"attr1","dataType":"text","logicalOperator":"=","connectorOperator":"AND","sequence":1,"expectedValue":"\"A\""},
    ...
  ]
}
```

**`LogicCondition` expected value**: comes from `Rule.RuleAttributes[attributeID].Value` of the winning variation.

**`EvaluateLogicConditions`**: reuses `evalSiblings` tree by converting `[]LogicCondition` to `[]entity.RuleCondition` stubs (only ID, Sequence, ParentID, ConnectorOp, LogicalOp needed for tree traversal). The `compareValues` call uses `userAttrs[attributeID]` as actual, `LogicCondition.ExpectedValue` as expected.

## Verification

**Commands:**
- `go test ./internal/service/evaluator/... -v -count=1` -- expected: all tests pass
- `go test ./internal/service/... -v -count=1` -- expected: all tests pass including new personalized delivery tests
- `go build ./...` -- expected: zero compile errors

## Spec Change Log
