# GORM DB Middleware Specification

## Overview

The `DBMiddleware` is a Gin middleware that injects a GORM database instance into the request context. This pattern ensures that every database operation performed within a request scope:

1.  Has a consistent **timeout** applied (preventing long-running queries from hanging).
2.  Is easily accessible via the context throughout the application layers.
3.  Supports proper **context propagation**, allowing for cancellation signal handling (e.g., if a client disconnects).

---

## Middleare Logic

For every incoming HTTP request, the middleware:

1.  Creates a new context derived from the request context with a predefined timeout.
2.  Attaches this new context to the global GORM DB instance using `db.WithContext(ctx)`.
3.  Stores the resulting "request-scoped" DB instance in:
    - **Gin Context**: For easy access in handlers using `c.Get()`.
    - **Standard Request Context**: For standard library compatibility and downstream GORM hooks.

---

## Configuration

| Constant    | Default Value | Description                                     |
| ----------- | ------------- | ----------------------------------------------- |
| `dbTimeout` | `10s`         | Maximum duration for any single DB transaction. |

---

## Usage

### 1. Router Setup

The middleware requires a pointer to the initialized `gorm.DB` instance.

```go
func SetupRouter(db *gorm.DB, ...) *gin.Engine {
    r := gin.Default()

    // Apply DB middleware globally
    r.Use(middleware.DBMiddleware(db))

    // ...
}
```

### 2. Retrieval in Handlers

#### Using GetDB Utility (Recommended)

The easiest way to retrieve the DB is to use the `ctxconsts.GetDB()` helper. This works with both `*gin.Context` and standard `context.Context`.

```go
func MyHandler(c *gin.Context) {
    db, ok := ctxconsts.GetDB(c.Request.Context()) // or simply c (since it implements context.Context)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "db not found"})
        return
    }

    // ... use db
}
```

#### Manual Retrieval (Legacy/Alternative)

---

## Context Keys

The middleware uses the key defined in `pkg/ctxconsts/ctxconsts.go`:

| Key Name          | Value  | Type                   |
| ----------------- | ------ | ---------------------- |
| `ctxconsts.DBKey` | `"DB"` | `ctxconsts.contextKey` |

---

## Implementation Files

| File                                                      | Purpose                   |
| --------------------------------------------------------- | ------------------------- |
| `internal/delivery/http/middleware/db_middleware.go`      | Middleware implementation |
| `internal/delivery/http/middleware/db_middleware_test.go` | Unit tests                |
| `pkg/ctxconsts/ctxconsts.go`                              | Context key definitions   |

---

## References

- [GORM Context Documentation](https://gorm.io/docs/context.html#Integration-with-Chi-Middleware)
