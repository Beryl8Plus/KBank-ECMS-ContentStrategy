# Permission Specification

## Overview

Access control is enforced via the `ProfilePermissionGuard` middleware.  
Each user belongs to a **Profile** (user group). A Profile is linked to zero or more **Permissions** through the `profile_permissions` junction table.  
A Permission is identified by a **source** (resource) and an **action** (operation).

The guard checks whether the authenticated user's active profile holds **at least one** of the required permissions (OR semantics). Edit / Delete actions apply to the user's own records only; Edit All / Delete All apply to all records in the system.

---

## Data Model

```
users
  └─ profile_id ──► profiles
                        └─ profile_permissions ──► permissions
                                                       ├─ source  (resource identifier)
                                                       └─ action  (operation identifier)
```

---

## Permission Sources

| Source Constant                | Value           | Description                    |
| ------------------------------ | --------------- | ------------------------------ |
| `PermissionSourceDecisionRule` | `decision_rule` | Content Decision Rule resource |

---

## Permission Actions

| Action Constant             | Value        | Scope | Description                             |
| --------------------------- | ------------ | ----- | --------------------------------------- |
| `PermissionActionCreate`    | `create`     | Own   | Create a new record                     |
| `PermissionActionEdit`      | `edit`       | Own   | Edit own records                        |
| `PermissionActionDelete`    | `delete`     | Own   | Delete own records                      |
| `PermissionActionViewAll`   | `view_all`   | All   | View all records across every BU / team |
| `PermissionActionEditAll`   | `edit_all`   | All   | Edit all records in the system          |
| `PermissionActionDeleteAll` | `delete_all` | All   | Delete all records in the system        |

---

## Permission Matrix — Content Decision Rule

| User Group                   | Create | Edit | Delete | View All | Edit All | Delete All |
| ---------------------------- | :----: | :--: | :----: | :------: | :------: | :--------: |
| Content Strategy Marker      |   ✅   |  ✅  |   ✅   |    ✅    |    ❌    |     ❌     |
| Content Strategy Super Admin |   ✅   |  ✅  |   ✅   |    ✅    |    ✅    |     ✅     |
| IT Admin                     |   ❌   |  ❌  |   ❌   |    ✅    |    ✅    |     ✅     |
| Viewer                       |   ❌   |  ❌  |   ❌   |    ✅    |    ❌    |     ❌     |

**Edit / Delete** — applies to the user's own records only.  
**Edit All / Delete All** — applies to all records in the system.  
**View All** — sees all rules from every BU / team.

---

## Middleware Usage

### Single action

```go
ProfilePermissionGuard(permRepo, enums.PermissionSourceDecisionRule, enums.PermissionActionCreate)
```

### Multiple actions (OR — any one grants access)

```go
ProfilePermissionGuard(permRepo, enums.PermissionSourceDecisionRule,
    enums.PermissionActionCreate,
    enums.PermissionActionEdit,
    enums.PermissionActionDelete,
)
```

### Router wire-up example

```go
decisionRule := r.Group("/decision-rules")
{
    // All authenticated profiles can view
    decisionRule.GET("",
        middleware.ProfilePermissionGuard(permRepo,
            enums.PermissionSourceDecisionRule,
            enums.PermissionActionViewAll),
        handler.ListDecisionRules,
    )

    // Only profiles with create permission
    decisionRule.POST("",
        middleware.ProfilePermissionGuard(permRepo,
            enums.PermissionSourceDecisionRule,
            enums.PermissionActionCreate),
        handler.CreateDecisionRule,
    )

    // Edit own OR edit all
    decisionRule.PUT("/:id",
        middleware.ProfilePermissionGuard(permRepo,
            enums.PermissionSourceDecisionRule,
            enums.PermissionActionEdit,
            enums.PermissionActionEditAll),
        handler.UpdateDecisionRule,
    )

    // Delete own OR delete all
    decisionRule.DELETE("/:id",
        middleware.ProfilePermissionGuard(permRepo,
            enums.PermissionSourceDecisionRule,
            enums.PermissionActionDelete,
            enums.PermissionActionDeleteAll),
        handler.DeleteDecisionRule,
    )
}
```

---

## HTTP Responses

| Condition                         | Status Code                 | Body                                 |
| --------------------------------- | --------------------------- | ------------------------------------ |
| `userID` not found in context     | `401 Unauthorized`          | `{"error": "unauthorized"}`          |
| `userID` in context is wrong type | `401 Unauthorized`          | `{"error": "unauthorized"}`          |
| Repository error                  | `500 Internal Server Error` | `{"error": "internal server error"}` |
| No matching permission found      | `403 Forbidden`             | `{"error": "forbidden"}`             |
| At least one permission matched   | `200` (passes through)      | —                                    |

---

## Context Key

The guard expects the user ID to be stored in the Gin context using the key `ctxconsts.UserIDKey` (value: `"userID"`) as a `uuid.UUID`, set by the upstream authentication middleware before this guard runs.

---

## Implementation Files

| File                                                                 | Purpose                          |
| -------------------------------------------------------------------- | -------------------------------- |
| `internal/domain/entity/enums/permission_action.go`                  | Source and action constants      |
| `internal/domain/repository/permission.go`                           | `PermissionRepository` interface |
| `internal/repository/permission_repository.go`                       | Postgres implementation          |
| `internal/delivery/http/middleware/profile_permission_guard.go`      | Guard middleware                 |
| `internal/delivery/http/middleware/profile_permission_guard_test.go` | Unit tests                       |
| `pkg/ctxconsts/ctxconsts.go`                                         | Context key definition           |
