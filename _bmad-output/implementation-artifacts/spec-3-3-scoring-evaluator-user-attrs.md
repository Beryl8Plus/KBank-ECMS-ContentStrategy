---
title: 'Scoring Evaluator — User-Attribute Parameter Threading'
type: 'refactor'
created: '2026-04-14'
status: 'done'
baseline_commit: 'feat/microservice-layer HEAD (post spec-3-2)'
context:
  - '_bmad-output/implementation-artifacts/spec-3-2-personalized-placement-evaluation.md'
---

## Intent

**Problem:** `evaluateSingleCondition` reads the "actual" attribute value from
`c.Attribute.Value` (DB-fetched static data). There is no way to pass live
user/customer attribute values through the `ScoringEvaluator` → `EvaluateRuleScore`
→ `evaluateConditionGroup` → `evaluateSingleCondition` chain, so personalized
evaluation must use a separate code path (`EvaluateLogicConditions` +
`evalLogicSiblings/evalLogicNode/evalLogicLeaf`) that duplicates the same tree-walk
logic.

**Approach:** Thread an optional `userAttrs map[string]json.RawMessage` parameter
through the internal evaluation chain (`evaluateConditionGroup`, `evalSiblings`,
`evalNode`, `evaluateSingleCondition`). Use a variadic signature on `EvaluateRuleScore`
so existing callers need no change. Add a `PersonalizedEvaluator` interface with
`EvaluateWithUserAttrs` implemented by `ScoringEvaluator`. Refactor
`EvaluateLogicConditions` to delegate to the same unified chain (building expected-value
map + Attribute DataType stub), removing the now-redundant
`evalLogicSiblings/evalLogicNode/evalLogicLeaf` functions.

## Boundaries & Constraints

**Always:**
- `EvaluateRuleScore(rule)` (zero additional args) continues to compile and behave
  identically to today — variadic signature provides backward compat.
- `ScoringEvaluator.Evaluate(ctx, rule)` signature unchanged — the `RuleEvaluator`
  interface does not change.
- `cms_runtime_service.go` requires no code changes — it calls `eval.Evaluate(ctx, rule)`
  which reaches `EvaluateRuleScore(rule)` (nil userAttrs = static DB path).
- `cms_delivery_service.go` requires no code changes — it calls
  `evaluator.EvaluateLogicConditions(entry.Conditions, userAttrs)` which internally now
  delegates to the unified chain.
- When `userAttrs` is `nil` (DB/runtime path), `evaluateSingleCondition` reads
  `c.Attribute.Value` exactly as it does today.
- When `userAttrs` is non-nil but does not contain the required attribute UUID,
  `evaluateSingleCondition` returns `false, nil` (same behaviour as current
  `evalLogicLeaf` for missing attrs — not an error, treated as non-match).
- `compareValues` signature unchanged.
- `logicConditionToRuleCondition` is updated in-place — no callers outside this package.

**Never:**
- Do not change the `RuleEvaluator` interface (`Evaluate` + `RuleType` only).
- Do not alter existing `EvaluateLogicConditions` public signature.
- Do not remove the `evalLogic*` functions until the unified path is verified
  green by tests (remove them in the same commit as the refactor).
- Do not push to remote or create a PR.

## I/O & Edge-Case Matrix

| Scenario | Input | Expected Behaviour |
|---|---|---|
| Static scoring (runtime) | `EvaluateRuleScore(rule)` / nil userAttrs | Reads `c.Attribute.Value`; behaviour unchanged |
| Personalized scoring — attr present, passes | `EvaluateRuleScore(rule, userAttrs)`, `userAttrs[attrID]` matches expected | Returns variation name + score |
| Personalized scoring — attr present, fails | `EvaluateRuleScore(rule, userAttrs)`, `userAttrs[attrID]` does not match expected | Returns nil variation + rule.Score |
| Personalized scoring — attr absent | `EvaluateRuleScore(rule, userAttrs)`, attr UUID not in userAttrs | `evaluateSingleCondition` returns false; variation skipped |
| No conditions | Any call with empty `rule.RuleConditions` | Returns nil, rule.Score, nil (unchanged) |
| `EvaluateWithUserAttrs` via registry hint | Caller type-asserts registry evaluator to `PersonalizedEvaluator` | Returns result from `EvaluateRuleScore(rule, userAttrs)` |
| `EvaluateLogicConditions` (unified path) | Conditions + userAttrs | Same result as before; `evalLogicLeaf` no longer involved |
| `EvaluateLogicConditions` — missing attr | `userAttrs` does not contain `lc.AttributeID` | Returns `false, nil` (unchanged contract) |

## Code Map

- `internal/service/evaluator/condition_evaluator.go` — core changes (evaluateSingleCondition, evalNode, evalSiblings, evaluateConditionGroup, EvaluateRuleScore, logicConditionToRuleCondition, EvaluateLogicConditions; remove evalLogicSiblings/evalLogicNode/evalLogicLeaf)
- `internal/service/evaluator/evaluator.go` — add `PersonalizedEvaluator` interface
- `internal/service/evaluator/scoring_evaluator.go` — add `EvaluateWithUserAttrs` method
- `internal/service/evaluator/condition_evaluator_test.go` (new) — tests for personalized path + `EvaluateWithUserAttrs` + unified `EvaluateLogicConditions`

## Tasks & Acceptance

**Execution order: 1 → 2 → 3 → 4**

- [ ] `internal/service/evaluator/condition_evaluator.go` — **Thread `userAttrs` through internal chain:**
  1. Change `evaluateSingleCondition(c, expectedValues, userAttrs map[string]json.RawMessage)`:
     - When `userAttrs != nil`: look up `userAttrs[c.AttributeID.String()]`; if present, use as `actualRaw`; if absent, return `false, nil`.
     - When `userAttrs == nil`: use `json.RawMessage(c.Attribute.Value)` as `actualRaw` (existing behaviour).
  2. Change `evalNode(…, userAttrs map[string]json.RawMessage)` — thread param through.
  3. Change `evalSiblings(…, userAttrs map[string]json.RawMessage)` — thread param through.
  4. Change `evaluateConditionGroup(conditions, expectedValues, userAttrs map[string]json.RawMessage)` — thread param through.
  5. Change `EvaluateRuleScore` to variadic: `EvaluateRuleScore(rule entity.DecisionRule, userAttrs ...map[string]json.RawMessage)`. Extract first element (or nil) and pass to `evaluateConditionGroup`.

- [ ] `internal/service/evaluator/evaluator.go` — **Add `PersonalizedEvaluator` interface:**
  ```go
  // PersonalizedEvaluator extends RuleEvaluator with user-attribute-aware evaluation.
  // Callers may type-assert a RuleEvaluator to PersonalizedEvaluator to use the
  // personalized path; if the assertion fails, fall back to Evaluate.
  type PersonalizedEvaluator interface {
      RuleEvaluator
      EvaluateWithUserAttrs(
          ctx context.Context,
          rule entity.DecisionRule,
          userAttrs map[string]json.RawMessage,
      ) (*string, float64, error)
  }
  ```
  Import `"encoding/json"`.

- [ ] `internal/service/evaluator/scoring_evaluator.go` — **Implement `EvaluateWithUserAttrs`:**
  ```go
  func (e ScoringEvaluator) EvaluateWithUserAttrs(
      _ context.Context,
      rule entity.DecisionRule,
      userAttrs map[string]json.RawMessage,
  ) (*string, float64, error) {
      return EvaluateRuleScore(rule, userAttrs)
  }
  ```
  Import `"encoding/json"`.

- [ ] `internal/service/evaluator/condition_evaluator.go` — **Unify `EvaluateLogicConditions` (remove `evalLogic*`):**
  1. Update `logicConditionToRuleCondition` to also set `rc.Attribute = &entity.Attribute{DataType: enums.AttributeDataType(lc.DataType)}`.
  2. Refactor body of `EvaluateLogicConditions`:
     - Build `expectedValues map[string]json.RawMessage` from `lc.AttributeID → lc.ExpectedValue`.
     - Build `rcs` using `logicConditionToRuleCondition` (now includes Attribute stub).
     - Build `byParent` + sort (same logic as before).
     - Call `evalSiblings(byParent, roots, 1, expectedValues, userAttrs)`.
  3. Delete `evalLogicSiblings`, `evalLogicNode`, `evalLogicLeaf`.

- [ ] `internal/service/evaluator/condition_evaluator_test.go` — **New test file:**
  - `TestEvaluateRuleScore_StaticPath` — existing static behavior unchanged (nil userAttrs).
  - `TestEvaluateRuleScore_PersonalizedPath_Pass` — user attr matches expected → returns variation.
  - `TestEvaluateRuleScore_PersonalizedPath_Fail` — user attr present but wrong value → nil variation.
  - `TestEvaluateRuleScore_PersonalizedPath_MissingAttr` — attr not in userAttrs → nil variation.
  - `TestScoringEvaluator_EvaluateWithUserAttrs` — verify `PersonalizedEvaluator` type assertion works and produces same result as `EvaluateRuleScore(rule, userAttrs)`.
  - `TestEvaluateLogicConditions_UnifiedPath` — regression: same input/output as before for `EvaluateLogicConditions`.

## Design Notes

**Why variadic on `EvaluateRuleScore`:**
Variadic `userAttrs ...map[string]json.RawMessage` lets all existing callers
(`ScoringEvaluator.Evaluate`, existing tests) compile without any change, while new
callers (tests, future delivery integrations) can pass `userAttrs` directly.

**Why `PersonalizedEvaluator` as a separate interface:**
The `RuleEvaluator` interface must remain stable (other evaluators may not support user
attrs). A separate `PersonalizedEvaluator` interface allows callers (e.g. future
delivery handlers that have both the rule entity and user attrs) to type-assert the
capability without breaking non-personalized evaluators.

**Why `EvaluateLogicConditions` is kept (not replaced by direct `EvaluateRuleScore` call):**
`EvaluateLogicConditions` operates on cached `[]LogicCondition` (no full DB entity
needed). `EvaluateRuleScore` requires `entity.DecisionRule` with preloaded `Attribute`
associations. The delivery path works from Redis cache, so `EvaluateLogicConditions`
remains the correct entry point for that path. However, its implementation is now
simplified by delegating to the unified `evaluateConditionGroup` chain.

**`evaluateSingleCondition` actual-value resolution logic:**
```
if userAttrs != nil:
    v, ok = userAttrs[c.AttributeID.String()]
    if ok:  actualRaw = v
    else:   return false, nil          // missing attr = non-match (not error)
else:
    actualRaw = json.RawMessage(c.Attribute.Value)   // DB path, unchanged
```

**`logicConditionToRuleCondition` Attribute stub:**
Only `DataType` is populated on the stub — `Value` remains its zero value. Since
`evaluateSingleCondition` only reads `c.Attribute.Value` when `userAttrs == nil`,
and `EvaluateLogicConditions` always passes a non-nil `userAttrs`, the zero-value
`Value` is never dereferenced on this path.

## Verification

```bash
go build ./internal/service/evaluator/...
go test ./internal/service/evaluator/...
go build ./...
```

Expected: 0 compile errors, all evaluator tests green, full build clean.
