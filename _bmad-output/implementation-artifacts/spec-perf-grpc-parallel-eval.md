---
title: 'Performance: Native Proto, Parallel Placement, Optimised Evaluator'
type: 'refactor'
created: '2026-04-22'
status: 'done'
baseline_commit: 'ec8ebe15bcea1ba06ca634d1612b7e48e119617c'
context:
  - proto/cms_runtime/v1/runtime.proto
  - internal/grpc/client/runtime_grpc_client.go
  - cmd/cms-runtime/internal/server/runtime_grpc_server.go
  - cmd/cms-delivery/service/cms_delivery_service.go
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The gRPC interface between cms-delivery and cms-runtime serialises schedules/user-attrs as JSON `bytes`, creating heavy Marshal/Unmarshal overhead on every request. Placements are processed sequentially in `GetPersonalizedContent`, and the condition evaluator re-parses JSON on every leaf comparison — all blocking the 250 TPS target.

**Approach:** (1) Replace `bytes schedules_json` / `bytes user_attrs_json` / `bytes logic_entries_json` with native Protobuf message fields so Protobuf binary encoding replaces JSON. (2) Process multiple placements concurrently via `errgroup`. (3) Pre-parse user-attr values once per request instead of `json.Unmarshal` on every condition leaf.

## Boundaries & Constraints

**Always:** Keep the `RuntimeEvaluator` interface signature stable (accept `[]*entity.Schedule` + `map[string]json.RawMessage`). Conversion to/from proto happens in the gRPC client/server layer only. Existing tests must continue to pass with updated proto types.

**Ask First:** If proto changes require a new `protoc` plugin or build step not already in the Makefile.

**Never:** Do not change the HTTP handler contract. Do not modify database schemas. Do not introduce new external dependencies beyond what is in go.mod.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Normal gRPC call | Schedules with rules + user attrs | Native proto messages round-trip correctly; identical ContentResult list | N/A |
| Empty user attrs | user_attrs map empty | Response with empty results (same as before) | N/A |
| Multiple placements | 5 placement names | All processed concurrently; results merged | First error cancels group |
| Single condition leaf | One Text EQ condition | Pre-parsed actual value compared without re-unmarshal | N/A |
| Nil evaluator (no gRPC) | evaluator == nil | Graceful skip, no panic | N/A |

</frozen-after-approval>

## Code Map

- `proto/cms_runtime/v1/runtime.proto` -- Proto definition; replace bytes fields with native messages
- `internal/grpc/pb/cms_runtime/v1/runtime.pb.go` -- Regenerated from proto
- `internal/grpc/pb/cms_runtime/v1/runtime_grpc.pb.go` -- Regenerated from proto
- `internal/grpc/client/runtime_grpc_client.go` -- Client; convert domain→proto on send, proto→domain on receive
- `cmd/cms-runtime/internal/server/runtime_grpc_server.go` -- Server; convert proto→domain on receive, domain→proto on send
- `cmd/cms-runtime/internal/server/runtime_grpc_server_test.go` -- Tests; update to use native proto types
- `cmd/cms-delivery/service/cms_delivery_service.go` -- Delivery service; add errgroup for parallel placement processing
- `cmd/cms-runtime/internal/evaluator/condition_evaluator.go` -- Evaluator; pre-parse user attrs to avoid repeated unmarshal

## Tasks & Acceptance

**Execution:**
- [ ] `proto/cms_runtime/v1/runtime.proto` -- Add native Schedule, UserAttr, LogicCondition messages and replace bytes fields with `repeated` native messages in request/response -- Eliminates JSON serialisation overhead
- [ ] Regenerate proto Go files -- Run `protoc` to generate new `.pb.go` files
- [ ] `internal/grpc/client/runtime_grpc_client.go` -- Add domain↔proto converters; update `Evaluate` to build/parse native proto messages -- Client-side serialisation elimination
- [ ] `cmd/cms-runtime/internal/server/runtime_grpc_server.go` -- Update `Evaluate` to receive/return native proto messages; convert to domain types internally -- Server-side serialisation elimination
- [ ] `cmd/cms-runtime/internal/server/runtime_grpc_server_test.go` -- Update test helpers to construct native proto request types
- [ ] `cmd/cms-delivery/service/cms_delivery_service.go` -- Wrap placement loop in `errgroup.Group` with bounded concurrency -- Parallel placement processing
- [ ] `cmd/cms-runtime/internal/evaluator/condition_evaluator.go` -- Add `ParsedUserAttrs` struct and pre-parse user attr values once; pass typed values to leaf comparators -- Remove per-leaf unmarshal

**Acceptance Criteria:**
- Given valid schedules and user attrs, when `cms-delivery` calls `cms-runtime` via gRPC, then no JSON marshal/unmarshal occurs on the gRPC wire path
- Given a request with multiple placements, when `GetPersonalizedContent` is called, then placements are evaluated concurrently
- Given a condition evaluator processing N leaf conditions, when evaluating, then user attr values are unmarshalled at most once per attribute per request
- Given all changes, when `go build ./...` and `go test ./...` are run, then both pass with zero failures

## Verification

**Commands:**
- `go build ./...` -- expected: no compilation errors
- `go test ./cmd/cms-runtime/...` -- expected: all tests pass
- `go test ./cmd/cms-delivery/...` -- expected: all tests pass
- `go test ./internal/grpc/...` -- expected: all tests pass
