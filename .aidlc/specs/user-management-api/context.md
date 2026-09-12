# Context Assessment

## Summary
<!-- 10-line max. Downstream phases read ONLY this section. -->
- **Type**: Brownfield
- **Scope**: new
- **Stack**: Go 1.26.4 / Fiber v2 / MongoDB (mongo-driver v2) / JWT (HMAC HS256) / gRPC / bcrypt
- **Architecture**: Hexagonal (Ports & Adapters) — domain / application / adapter / infrastructure
- **Feature**: RESTful API สำหรับจัดการผู้ใช้ (register, login, CRUD) ด้วย Go, MongoDB สำหรับ persistence และ JWT สำหรับ authentication
- **Impact**: New standalone
- **Complexity**: Medium — 13 stories (7 core + 6 bonus), 1 domain, 1 user type
- **Recommendations**: Personas No, Units No, NFR Yes

## Project Overview
- **Type**: Brownfield — โค้ดผลเฉลย Challenge ครบทุก requirement พร้อม test suite
- **Assessment Date**: 2026-09-13T02:09:43+0700

## Technology Stack
- **Languages**: Go 1.26.4
- **Frameworks**: Fiber v2 (HTTP), google.golang.org/grpc (gRPC), oapi-codegen (OpenAPI 3.0.3 codegen), golang-jwt/v5, mongo-driver v2
- **Build System**: Makefile (`make build` / `run` / `test` / `codegen` / `proto` / `mock` / `docsgen`)
- **Testing**: Go standard `testing` package + go.uber.org/mock (gomock) + Fiber `app.Test` + opt-in MongoDB integration test
- **Infrastructure**: Docker + docker-compose (api + mongo:7 + healthcheck), config ผ่าน YAML + env override + secret files

## Patterns & Conventions

- **Architecture pattern**: Hexagonal (Ports & Adapters) — ลำดับชั้น `adapter → application → domain`, outbound adapters อยู่ใต้ domain ports
- **Data access**: Repository pattern ผ่าน interface port — domain กำหนด `Repository` interface, MongoDB adapter implement ไว้ที่ `internal/adapter/outbound/mongodb/`
- **API response format**: JSON ตรงจาก domain/application DTOs; ไม่มี envelope — success คืน payload ตรง, error คืน `{ "error": "..." }` ผ่าน `WriteServiceError`
- **Error handling**: domain sentinel errors (`ErrNotFound`, `ErrDuplicateEmail`, `ErrInvalidCredentials`) + `ValidationError`; จับคู่เป็น HTTP/gRPC codes ใน layer ของ adapter
- **Authentication**: JWT HMAC `HS256` ใน `Authorization: Bearer` header; middleware `Authenticate` สำหรับ guarded endpoints; gRPC ใช้ token metadata interceptor
- **Validation**: application layer (`normalizeEmail`, `validEmail`, password length ≥ 8, body-size limit) คืน `ValidationError`
- **Logging**: Go `log/slog`; logging middleware บันทึก method, path, status, duration; background task log ผ่าน `countUsersLoop`
- **Code style**: gofmt; ตั้งชื่อ package ตาม Go idiom; generated code ถูกแยกไว้ (`server_openapi.gen.go`, `*.pb.go`, `internal/mocks/`) และไม่นับ coverage

## Codebase Analysis

**Entry Points**:

| Entry Point | Type | Description |
|-------------|------|-------------|
| `cmd/api/main.go` | Composition root | ประกอบ Fiber server + gRPC server + MongoDB + JWT/bcrypt + background `countUsersLoop`; graceful shutdown (`signal.NotifyContext`, `app.Shutdown`, `GracefulStop`) |

**Module Dependencies**:
```
adapter/inbound (http + grpc)
        ▼
application/services/user, statusdb  (use cases)
        ▼        ▼
application/ports (hasher, token)   domain/{user,statusdb} (entities + repository ports)
        ▼
adapter/outbound (mongodb, jwt, bcrypt) → infrastructure/db (MongoDB Connect)
```

**Key Abstractions**:

| Abstraction | Location | Purpose |
|-------------|----------|---------|
| `user.Repository` | `internal/domain/user/repository.go` | Domain port สำหรับ user persistence (Create/FindByID/FindByEmail/List/Update/Delete) |
| `statusdb.Repository` | `internal/domain/statusdb/repository.go` | Domain port สำหรับ collection stats (`Count`) |
| `ports.Hasher` / `ports.TokenManager` | `internal/application/ports/` | Application ports สำหรับ bcrypt + JWT |
| `user.Service` interface | `internal/application/services/user/` | Use case surface (Register/Login/List/GetUser/UpdateUser/DeleteUser/Count) |
| `ServerInterface` | `internal/adapter/inbound/http/server_openapi.gen.go` | oapi-codegen กำหนด handler ให้ Fiber server implement |
| `Responder` | `internal/adapter/inbound/http/user/handler.go` | DecodeBody + WriteServiceError plumbing |

**Data Flow**:
```
HTTP:  Fiber + oapi-codegen → middleware.Authenticate (JWT) → user.Handler → user.Service → domain ports → mongodb repo → MongoDB
gRPC:  grpc.Server → tokenAuth interceptor → grpc server (userv1.UserServiceServer) → user.Service → domain ports → MongoDB
```

**Integration Points**:
- MongoDB (`users` collection, unique email index) — persistence
- JWT secret (HMAC HS256) — token issuance/validation
- gRPC listen `GRPC_PORT` (default 50051) — inbound adapter ขนานกับ HTTP

## Feature Impact

**Affected Areas**: New standalone — โปรเจกต์นี้เป็น product ทั้งโปรเจกต์ ครอบคลุมทั้ง 7 core requirements + 6 bonus

| Area | Impact | Reason |
|------|--------|--------|
| `internal/domain/` | Existing (สมบูรณ์) | User entity + business rules + repository ports |
| `internal/application/` | Existing (สมบูรณ์) | Use cases: auth, CRUD, count |
| `internal/adapter/` | Existing (สมบูรณ์) | Inbound HTTP/gRPC + outbound mongodb/jwt/bcrypt + middleware |
| `internal/infrastructure/db` | Existing (สมบูรณ์) | MongoDB connect wrapper |
| `api/` | Existing (สมบูรณ์) | `openapi.yaml` (source of truth), `codegen.yaml`, `user_service.proto` |
| `config/`, `Makefile`, `Dockerfile`, `docker-compose.yml` | Existing (สมบูรณ์) | Config, build/test/codegen tooling, containerization |
| `docs/`, `documents/` | Existing (สมบูรณ์) | OpenAPI bundle + requirement/design documents |

## Recommendations

- Story Count: Medium (13) — 7 core (model, auth, CRUD, MongoDB, middleware, concurrency, testing) + 6 bonus
- Domain Boundaries: 1 domain — user management (user + statusdb collection stats)
- User Types: 1 persona — API consumer / backend integrator
- Integration Points: MongoDB (persistence), JWT, gRPC — ทั้งหมดเป็น internal ให้เฉพาะ feature นี้
- **Personas**: No — มีผู้ใช้ระบบแค่แบบเดียว (API consumer) ไม่คุ้มกับการทำ persona profile
- **Units**: No — ขอบเขตเดียว ครอบคลุมกัน (auth + CRUD + concurrency ใช้ domain เดียวกัน) แบ่งหน่วยจะเพิ่ม overhead
- **NFR**: Yes — security (JWT, hashing), performance (query/index, background count task), testability (interface abstraction) ถูกประเมินจากเกณฑ์

## Scope

- **Detected scope**: new
- **Rationale**: Standalone product ที่ถูกกำหนดด้วย requirements ครบถ้วน (สร้างจากชุดเอกสาร requirement) — ต้องการ full workflow; ถึงแม้โปรเจกต์นี้มีโค้ดปัจจุบัน แต่ scope covers การสร้าง product ทั้งหมด
- **Phases skipped**: None — full workflow (context → requirements → decomposition → design → tasks → implement → build → deploy)

## Recommended Workflow

```
┌─────────────┐
│   Context ✅ │
└──────┬──────┘
       ▼
┌─────────────┐
│ Requirements ✅ │
└──────┬──────┘
       ▼
┌─────────────┐
│    Design ✅  │
└──────┬──────┘
       ▼
┌─────────────┐
│    Tasks ✅   │
└──────┬──────┘
       ▼
┌─────────────┐
│  Implement 🔄 │
└──────┬──────┘
       ▼
┌─────────────────────┐
│  Build and Test     │
└──────┬──────────────┘
       ▼
┌─────────────┐
│   Deploy    │
└─────────────┘
```

## External References

| Source | Type | What was used |
|--------|------|---------------|
| `documents/requirements/README.md` | Requirement summary | โครงสร้าง 2 Paths, เลือก Path 1 (User Management API) |
| `documents/requirements/user-management-api/requirement.md` | Requirement spec | 7 core requirements + 6 bonus + deliverables + evaluation criteria |
| `documents/requirements/user-management-api/requirement_brd.md` | Implementation BRD | สถานะตาม requirement, deep-dive concurrency task, gRPC design |
| `documents/requirements/user-management-api/tech-stack.md` | Tech stack / architecture | โครงสร้างโปรเจกต์, hexagonal layering, DI, tooling |