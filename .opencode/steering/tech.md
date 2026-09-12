# Steering: Tech

## Summary
- Stack: Go 1.26.4 / Fiber v2 / mongo-driver v2 / JWT (HMAC HS256) / gRPC / bcrypt
- Architecture/API style: Hexagonal (ports & adapters) · REST (OpenAPI 3.0.3, oapi-codegen) + gRPC
- Infra: Docker + docker-compose (api + mongo:7 + healthcheck)

## Stack
- **Languages**: Go 1.26.4
- **Frameworks**: Fiber v2, google.golang.org/grpc, oapi-codegen (OpenAPI 3.0.3 codegen), golang-jwt/v5, go.mongodb.org/mongo-driver/v2
- **Build system**: Makefile targets (build / run / test / test-integration / tools / codegen / proto / mock / docsgen)
- **Package manager**: Go modules (`go.mod` / `go.sum`)
- **Testing**: standard `testing`, go.uber.org/mock (gomock), Fiber `app.Test`, opt-in live MongoDB integration test

## Architecture
- **Pattern**: Hexagonal (Ports & Adapters) — inbound adapters (HTTP Fiber, gRPC) → application (services/use cases) → domain (entities + repository ports) ← outbound adapters (mongodb, jwt, bcrypt); infra (`db`) provides MongoDB connect
- **API style**: REST (OpenAPI 3.0.3 source of truth at `api/openapi.yaml`, server gen via oapi-codegen) + gRPC (`api/proto/user_service.proto`)

## Infrastructure
- **Cloud provider**: N/A (local/containerized deliverable)
- **Compute**: binary `bin/api_user_mgr`; Docker container
- **Database**: MongoDB (mongo:7) — `users` collection with unique email index
- **IaC tool**: N/A (docker-compose for local orchestration)

## Patterns & Conventions
- **Architecture pattern**: Hexagonal, dependency direction `adapter → application → domain`; outbound adapters implement domain ports
- **Data access**: Repository pattern via domain port interfaces; MongoDB adapter at `internal/adapter/outbound/mongodb/`
- **API response format**: direct JSON models from application DTOs; success = payload, error = `{ "error": "..." }` via `WriteServiceError`
- **Error handling**: domain sentinel errors + `ValidationError`; mapped to HTTP/gRPC codes at adapter layer
- **Authentication**: JWT HMAC HS256, `Authorization: Bearer`, `middleware.Authenticate`; gRPC token-metadata interceptor
- **Validation**: application-layer (`validEmail`, `normalizeEmail`, password ≥ 8 chars, body-size limit)
- **Logging**: Go `log/slog` (text handler); logging middleware logs method/path/status/duration; `countUsersLoop` logs user totals
- **Code style**: gofmt; short idiomatic Go naming; generated files isolated (`*.gen.go`, `*.pb.go`, `internal/mocks/`) and excluded from coverage
- **Naming conventions**: layered packages (`domain`, `application`, `adapter`, `infrastructure`); port files named `repository.go`, `token.go`, `hasher.go`
- **Branch strategy**: no multi-branch workflow in this repo

## Environment Configuration
- **Config approach**: `config.Load()` — YAML base + secret file lookup + env override (per-environment YAML + `secret_dev.go` / `secret_prod.go` build tags)
- **Environments**: dev / prod (`APP_ENV`)
- **Secrets management**: secret files per env via Go files; JWT secret passed at runtime

## CICD Pipeline
- **Tool**: none configured — local `make` targets + Docker
- **Stages**: build → test → coverage → integration (opt-in `TEST_MONGO_URI`)
- **Deploy target**: container runtime (docker-compose)

## Dependency Management
- **Lockfile**: `go.sum`
- **Version strategy**: direct pinned major versions in `go.mod`
- **Monorepo tooling**: N/A

## Known Technical Debt
- `EstimatedDocumentCount` may under-count after deletions (accepted for logging job; `CountDocuments` + `_id_` hint kept as documented fallback)
- `internal/adapter/middleware/recovery_middleware.go` is unused — server relies on Fiber built-in recover
- gRPC stubs and OpenAPI handler are generated; require `make proto` / `make codegen` + toolchain when regenerating