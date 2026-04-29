# KBank ECMS

## Description

This repository exposes the Rule Management API as a Go backend service structured following [golang-standards/project-layout](https://github.com/golang-standards/project-layout) with Clean Architecture principles.

- `POST /rule-management`: active API endpoint.

### Core Architectural Layers

The project is organized into four clean layers with a strict inward dependency rule — outer layers depend on inner layers, never the reverse.

```mermaid
flowchart TD
    Entry[cmd/server/main.go<br>Entry Point & Wiring]

    subgraph Delivery [Delivery Layer]
        Http[internal/delivery/<br>http layer]
    end

    subgraph Service [Service Layer]
        BizLogic[internal/service/<br>business logic]
    end

    subgraph Repo [Repository Layer]
        DataAccess[internal/repository/<br>data access impl]
    end

    subgraph Domain [Domain Layer - Core zero deps]
        Core[internal/domain/]
    end

    Entry --> Delivery
    Entry --> Service
    Entry --> Repo

    Delivery --> Domain
    Service --> Domain
    Repo --> Domain
```

#### Layer Responsibilities

| Layer              | Path                              | Responsibility                                               |
| ------------------ | --------------------------------- | ------------------------------------------------------------ |
| **Domain**         | `internal/domain/`                | Entities, repository interfaces. No external dependencies.   |
| **Use Case**       | `internal/usecase/`               | Business logic orchestration. Depends on domain only.        |
| **Repository**     | `internal/repository/`            | Redis & Azure implementations. Implements domain interfaces. |
| **Delivery**       | `internal/delivery/http/`         | Gin HTTP handlers, middleware, route definitions.            |
| **Infrastructure** | `internal/infrastructure/logger/` | Structured logging (cross-cutting concern).                  |
| **Pkg**            | `pkg/util/`                       | Generic public utilities safe for external use.              |

#### Project Structure

```text
├── cmd/
│   └── server/
│       └── main.go                          # Entry point — wires all layers
├── internal/
│   ├── domain/                              # Layer 1: Core entities & interfaces
│   ├── usecase/                             # Layer 2: Business logic
│   ├── repository/                          # Layer 3: Data access implementations
│   ├── delivery/                            # Layer 4: HTTP delivery
│   └── infrastructure/
│       └── logger/                          # Structured logging
├── pkg/
│   └── util/                                # Generic utilities
├── configs/                                 # YAML configuration files
├── docs/                                    # API docs & diagrams
└── dockerfile
```

## Installation

### Prerequisites

- [Go](https://golang.org/) 1.26+ (or your relevant Go version)
- [Docker](https://www.docker.com/) (Optional: for containerized deployment)
- [Redis](https://redis.io/)

### Authentication

The API uses JWT (JSON Web Token) authentication following Gin Framework standards:

**Public Endpoints** (no authentication required):
- `POST /rule-management`
- `POST /token` (OAuth2 token endpoint for client credentials)

**Protected Endpoints** (JWT authentication required):
- All endpoints under `/schedules`, `/decision-rules`, `/schedule-occurrences`, `/attributes`, `/channels`, `/placements`

#### User Authentication Flow
1. Include JWT token in Authorization header: `Authorization: Bearer <token>`
2. Middleware validates token before allowing access to protected endpoints
3. User information is extracted and stored in context for downstream handlers

#### OAuth2 Client Credentials Flow (Server-to-Server)
For server-to-server communication, the API supports OAuth2 Client Credentials Flow:

**Client Server Implementation Flow:**

1. **Boot up**: Server starts (no token in memory)
2. **Request API**: When needing data, check memory cache for existing token
3. **If no token/expired**: Call `/token` endpoint to obtain new token, cache it
4. **Call Service**: Attach token in Authorization header and call API
5. **Handle 401**: If API returns 401, clear cache and retry from step 2

**1. Obtain Token:**
```bash
curl -X POST "http://localhost:8081/token" \
  -d "grant_type=client_credentials" \
  -d "client_id=service-cmc" \
  -d "client_secret=super-secret-key-cmc"
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 86400
}
```

**2. Use Token for API Requests:**
```bash
curl -X GET "http://localhost:8081/schedules" \
  -H "Authorization: Bearer <access_token>"
```

**Supported Clients:**
Clients are configured in `configs/oauth2-clients.yaml`. Current clients:
- `service-cmc`: Scopes: `read:rules`, `write:rules`

**Adding New Clients:**
To add a new OAuth2 client without modifying code:

1. Edit `configs/oauth2-clients.yaml`:
```yaml
clients:
  - client_id: your-new-service
    client_secret: your-secret-key-here
    scopes:
      - read:orders
      - write:payments
    description: Your new service description
```

2. (Optional) Override secret in `.env` for security:
```bash
CLIENT_SECRET_your-new-service=production-secret-from-vault
```

3. Restart the application to load the new configuration

**Best Practices:**
- **Token Caching**: Store tokens in memory with TTL to avoid unnecessary `/token` calls
- **Scope-Based Access**: Use scopes to limit what each client can do
- **Rate Limiting**: Implement rate limiting per client_id to prevent abuse
- **Secret Rotation**: Regularly rotate client secrets and support revocation
- **Error Handling**: Clear cache on 401 and retry token acquisition
- **External Config**: Use `configs/oauth2-clients.yaml` for client definitions, override secrets via environment variables in production

**JWT Configuration** (`.env`):
```bash
JWT_SECRET_KEY=your-secret-key-change-in-production
JWT_TOKEN_DURATION=24h

# OAuth2 Client Configuration
OAUTH2_CLIENTS_CONFIG_PATH=configs/oauth2-clients.yaml

# Override client secrets from environment variables (optional, for security)
# Format: CLIENT_SECRET_<CLIENT_ID>
CLIENT_SECRET_service-cmc=super-secret-key-cmc
```

**Authentication Sequence Diagram:**

```mermaid
sequenceDiagram
    participant ClientServer
    participant Cache
    participant API
    participant TokenHandler
    participant JWTMiddleware
    participant Handler

    Note over ClientServer,Handler: OAuth2 Client Credentials Flow (Server-to-Server)

    Note over ClientServer: 1. Boot up (no token)
    ClientServer->>Cache: Check for token
    Cache-->>ClientServer: No token found

    Note over ClientServer: 2. Request API (first time)
    ClientServer->>Cache: Check for token
    Cache-->>ClientServer: No token / Expired

    Note over ClientServer: 3. Obtain new token
    ClientServer->>API: POST /token<br/>grant_type=client_credentials<br/>client_id=service-cmc<br/>client_secret=xxx
    API->>TokenHandler: Validate credentials
    TokenHandler->>TokenHandler: Check client registry
    TokenHandler->>TokenHandler: Generate JWT token
    TokenHandler-->>API: access_token + expires_in
    API-->>ClientServer: 200 OK + access_token

    ClientServer->>Cache: Store token with TTL

    Note over ClientServer: 4. Call service with token
    ClientServer->>Cache: Check for token
    Cache-->>ClientServer: Token found
    ClientServer->>API: GET /schedules<br/>Authorization: Bearer <token>
    API->>JWTMiddleware: Extract token
    JWTMiddleware->>JWTMiddleware: Verify client token
    JWTMiddleware->>JWTMiddleware: Extract client_id + scopes
    JWTMiddleware->>Handler: c.Next()
    Handler->>API: Process request
    API-->>ClientServer: 200 OK + Data

    Note over ClientServer: 5. Handle 401 (token expired)
    ClientServer->>API: GET /schedules<br/>Authorization: Bearer <expired_token>
    API->>JWTMiddleware: Verify token
    JWTMiddleware-->>API: Token expired
    API-->>ClientServer: 401 Unauthorized
    ClientServer->>Cache: Clear token cache
    ClientServer->>API: POST /token<br/>(retry token request)
    API-->>ClientServer: New access_token
    ClientServer->>Cache: Store new token
    ClientServer->>API: GET /schedules<br/>Authorization: Bearer <new_token>
    API-->>ClientServer: 200 OK + Data

    Note over ClientServer,Handler: User Authentication Flow

    participant User
    User->>API: GET /schedules<br/>Authorization: Bearer <user_token>
    API->>JWTMiddleware: Extract token
    JWTMiddleware->>JWTMiddleware: Verify user token
    JWTMiddleware->>JWTMiddleware: Extract user_id + email
    JWTMiddleware->>Handler: c.Next()
    Handler->>API: Process request
    API-->>User: 200 OK + Data

    Note over ClientServer,Handler: Public Endpoint (No Auth)

    ClientServer->>API: POST /rule-management
    API->>Handler: Process directly
    API-->>ClientServer: 200 OK + Data
```

### Build

This project utilizes a `Makefile` to simplify common build and development tasks. You can build the project for local testing or containerization using the following commands:

```bash
# Initialize workspace (install linters, swag, goose, and git hooks)
make init

# Local build (outputs binary to bin/server)
make build

# Build Docker Image (tags as kbank-ems:latest)
make dev-build
```

### Local Run

To set environment values and run the server locally, execute the following commands:

**Windows (PowerShell)**

```powershell
$env:SETENV="DEVLOCAL"
$env:REDIS_HOST="localhost"
$env:REDIS_PORT="6379"
go run ./cmd/server/
```

**Unix/macOS**

```bash
SETENV=DEVLOCAL REDIS_HOST=localhost REDIS_PORT=6379 go run ./cmd/server/
```

Upon successful execution, the service will start listening on `:8081`.

### Docker Compose

For the local container stack, start the services with Docker Compose:

```bash
docker compose up -d
```

Local endpoints:

- Rule Management API: `http://localhost:8081`
- CMS Delivery API: `http://localhost:8082`
- Swagger UI: `http://localhost:8083`
- RedisInsight: `http://localhost:5540`

RedisInsight is preconfigured to connect to the Compose Redis service as `local-redis`.

## Usage

You can test the active API endpoint by making an HTTP request. Below is an example using `curl`:

```bash
curl -X POST "http://localhost:8081/rule-management" \
  -H "requestID: req-002" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Configuration Files

Environment variables and configurations are read from properties in the `configs/` directory during startup:

- `configs/newservice_inbound_config.yaml` — Inbound rate limit & server settings
- `configs/newservice_outbound_config.yaml` — Outbound service settings
- `configs/redis_config.yaml` — Redis connection configurations

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)
