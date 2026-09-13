# Steering: Structure

## Summary
- Repo type: Single Go module — complete challenge deliverable (API + specs + docs)
- Key source dirs: `cmd/`, `internal/{domain,application,adapter,infrastructure}`, `api/`, `config/`
- Main entry points: `cmd/api/main.go` (Fiber + gRPC composition root)

## Repository
- **Type**: Single — one Go module `github.com/ArchangleT/7-solutions-backend-challenge`
- **Root**: module root contains Makefile, go.mod/go.sum, Dockerfile, docker-compose.yml, OpenAPI/proto specs, docs, documents

## Key Directories

| Directory | Purpose | Key Contents |
|-----------|---------|--------------|
| `cmd/api/` | Composition root | `main.go` — Fiber + gRPC wiring, countUsersLoop, graceful shutdown |
| `internal/domain/user/` | Domain core | `entity.go`, `errors.go`, `repository.go` (port) |
| `internal/domain/statusdb/` | Stats port | `repository.go` (`Count`) |
| `internal/application/ports/` | App ports | `hasher.go`, `token.go` |
| `internal/application/services/user/` | Use cases | `service.go`, `dto.go`, `utils.go`, tests |
| `internal/application/services/statusdb/` | Count use case | `service.go`, tests |
| `internal/adapter/middleware/` | Fiber middleware | `auth_middleware.go`, `log_middleware.go`, `recovery_middleware.go` |
| `internal/adapter/inbound/http/` | HTTP inbound | `server.go`, `server_openapi.gen.go`, `user/handler.go`, `health-check/`, `helpers/utils.go` |
| `internal/adapter/inbound/grpc/` | gRPC inbound | `server.go`, `auth.go`, `userv1/*.pb.go` |
| `internal/adapter/outbound/mongodb/` | Mongo adapter | `user_repository.go` + unit/integration tests |
| `internal/adapter/outbound/jwt/`, `bcrypt/` | Outbound ports impl | token manager, password hasher |
| `internal/infrastructure/db/` | DB connect | `mongodb.go` + tests |
| `internal/mocks/` | Generated mocks | `statusdb/`, `userrepo/`, `ports/` |
| `api/` | Specs (source of truth) | `openapi.yaml`, `codegen.yaml`, `proto/user_service.proto` |
| `config/` | Config package | `config.go`, `config_dev.yaml`, `secret_dev.go`, `secret_prod.go` |
| `docs/` | Bundled OpenAPI | `swagger.json` / `swagger.yaml` |
| `documents/` | Requirements/design docs | `requirements/` (README + per-path docs) |

## Key Files

| File | Purpose | Notes |
|------|---------|-------|
| `api/openapi.yaml` | API contract source of truth | oapi-codegen input |
| `api/proto/user_service.proto` | gRPC spec | CreateUser / GetUser |
| `cmd/api/main.go` | Dependency injection + lifecycle | composition root |
| `Makefile` | Tooling | build/run/test/codegen/proto/mock/docsgen |
| `docker-compose.yml` | Local orchestration | api + mongo:7 + healthcheck |
| `go.mod` | Module definition | Go 1.26.4 |

## Entry Points

| Entry | Type | Description |
|-------|------|-------------|
| `cmd/api/main.go` | API server | Starts Fiber HTTP server + gRPC server + background count task; graceful shutdown on SIGINT/SIGTERM |

## Module Dependencies

```
adapter/inbound/http ──► application/services/user ──► application/ports (hasher, token)
adapter/inbound/grpc ──► application/services/user ──► domain/user (Repository port)
application/services/statusdb ──► domain/statusdb (Repository port)
adapter/outbound/mongodb ──► domain/user + domain/statusdb (implements ports) ──► infrastructure/db
adapter/outbound/jwt, bcrypt ──► application/ports (implements ports)
```

- **Dependency rule**: domain knows nothing about Fiber/HTTP/MongoDB; adapters depend on ports; composition happens only at `cmd/api/main.go`.

## Data Flow

```
HTTP:  Fiber + oapi-codegen → middleware.Authenticate (JWT) → user.Handler (Responder) → user.Service → Repository port → mongodb adapter → MongoDB
gRPC:  grpc.Server → tokenAuth interceptor → userv1.UserServiceServer → user.Service → Repository port → mongodb adapter → MongoDB
Count: countUsersLoop (10s ticker) → statusdb.Service → statusdb.Repository port → mongodb adapter.EstimatedDocumentCount
```

## Key Abstractions

| Abstraction | Location | Purpose | Used By |
|-------------|----------|---------|---------|
| `user.Repository` | `internal/domain/user/repository.go` | User persistence port | `user.Service`, mongodb adapter |
| `statusdb.Repository` | `internal/domain/statusdb/repository.go` | Collection stats port | `statusdb.Service`, mongodb adapter |
| `ports.Hasher` | `internal/application/ports/hasher.go` | Password hashing port | `user.Service` |
| `ports.TokenManager` | `internal/application/ports/token.go` | JWT sign/parse port | `user.Service`, auth middleware, gRPC auth |
| `user.Service` | `internal/application/services/user/` | Use case facade | HTTP handlers, gRPC server |
| `ServerInterface` | `internal/adapter/inbound/http/server_openapi.gen.go` | HTTP route contract | fiber server |
| `Responder` | `internal/adapter/inbound/http/user/handler.go` | Decode/encode + error writing | user handlers |

## Test Organization
- **Location**: co-located `_test.go` beside source; mocks in `internal/mocks/` (gomock)
- **Types present**: table-driven unit tests (service/utils/jwt/bcrypt/config/health), Fiber `app.Test` handler tests, bufconn in-process gRPC tests, opt-in live MongoDB integration test (`TestConnect_Live`, `TEST_MONGO_URI`)
- **Utilities**: fake in-memory repo for service tests; gomock for statusdb/ports/userrepo
- **Coverage**: `make test` runs `go test ./... -coverprofile`, excludes `internal/mocks/`, `server_openapi.gen.go`, `*.pb.go`
- **Run command**: `make test` (unit), `make test-integration` (live, needs `TEST_MONGO_URI`)

## Build & Deploy
- **Build output**: `bin/api_user_mgr` (`make build`; `go build -o bin/api_user_mgr ./cmd/api`)
- **Container**: `Dockerfile` (multi-stage go build → runtime), `docker-compose.yml` (api + mongo:7 + healthcheck)
- **Deploy target**: local containerized runtime via docker-compose