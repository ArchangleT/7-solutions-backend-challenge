# Design Document: User Management API

## Summary
- **Architecture**: Hexagonal (Ports & Adapters) — single binary monolith, dependency `adapter → application → domain`
- **Stack**: Go 1.26.4 / Fiber v2 + oapi-codegen (REST) / gRPC / MongoDB (mongo-driver v2) / JWT (HMAC HS256) / bcrypt / slog
- **Components**: 12 — Domain·User, User Service, Hasher port, Token port, HTTP Adapter, gRPC Adapter, Mongo Adapter, JWT manager, bcrypt hasher, Logging middleware, Count loop, Composition root
- **Entities**: 1 — User
- **Endpoints**: 7 REST (`/health`, register, login, list, get, update, delete) + 2 gRPC (`CreateUser`, `GetUser`)
- **Integrations**: MongoDB (persistence), JWT (in-process secret), gRPC (inbound channel)
- **Testing**: PBT No — table-driven unit + opt-in live integration + bufconn gRPC · NFR Yes
- **Key Decisions**: Fiber v2 (existing stack), hybrid REST+gRPC, OpenAPI oapi-codegen, custom validation, sentinel errors, official mongo-driver v2, no caching MVP, JWT HS256, bcrypt, YAML+env+secrets

## Architecture

### System Context Diagram
```
┌───────────────────────────────  Application  ───────────────────────────────┐
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐    ┌───────────────────────────────┐   │
│  │ HTTP client  │  │ gRPC client  │    │ countUsersLoop (10s ticker)   │   │
│  └──────┬───────┘  └──────┬───────┘    └──────────────┬────────────────┘   │
│         ▼                ▼                             ▼                    │
│  ┌──────────────────────────────────────────────────────────────────┐      │
│  │ Inbound: Fiber (oapi-codegen) + middleware  ·  grpc.Server      │      │
│  └──────────────────────────────┬───────────────────────────────────┘      │
│                                 ▼                                         │
│  ┌──────────────────────────────────────────────────────────────────┐      │
│  │ Application: user.Service (Register/Login/CRUD) · statusdb       │      │
│  └───────┬──────────────────────────┬───────────────────────────────┘      │
│          │  ports (Hasher/Token)    │  ports (user.Repository, statusdb.Repo)│
│  ┌───────▼───────────────────┐  ┌───▼───────────────────────────────┐       │
│  │ Outbound: jwt · bcrypt    │  │ Domain: User entity · rules        │      │
│  └───────────────────────────┘  └───┬───────────────────────────────┘      │
│                                     │ Repository (implemented by adapter)  │
│  ┌──────────────────────────────────▼───────────────────────────────┐      │
│  │ Outbound: mongodb.UserRepository (official driver v2)            │      │
│  └──────────────────────────────────────────────────────────────────┘      │
└──────────────────────────────│─────────────────────────────────────────────┘
                               ▼
                     ┌──────────────────┐
                     │     MongoDB       │  (users collection + email_1 unique)
                     └──────────────────┘
```

### Technology Stack
- **Frontend**: none (backend-only challenge)
- **Backend**: Go 1.26.4 · Fiber v2 (HTTP) · google.golang.org/grpc (gRPC) · oapi-codegen (OpenAPI 3.0.3 codegen)
- **Database**: MongoDB (driver: `go.mongodb.org/mongo-driver/v2`) — collection `users`
- **Infrastructure**: Docker + docker-compose (api + mongo:7 + healthcheck); config YAML+env+secret files
- **Key Libraries**: golang-jwt/jwt/v5 (HS256), golang.org/x/crypto bcrypt, go.uber.org/mock (test)

### Key Design Decisions
1. **Hexagonal monolith** (single binary): Domain ไม่รู้จัก framework/driver; adapters implement ports; composition ที่ `main.go` เท่านั้น → สลับ technologies ได้โดยไม่แตะ business logic
2. **Official mongo-driver v2 + unique email index**: persistence ตาม requirement; unique index `email_1` เป็น defense กัน race duplicate email
3. **Hybrid inbound (REST + gRPC)**: `CreateUser`/`GetUser` เปิดผ่าน gRPC ขนานกับ REST โดยใช้ `user.Service` + `ports.TokenManager` ร่วมกัน; gRPC auth ผ่าน token metadata interceptor

## Traceability

| Requirement | Component(s) | API Endpoint(s) | Data Entity | Design File | Status |
|---|---|---|---|---|---|
| US-001 User Model | C1 Domain·User | — | User | components, data-model | ✅ Covered |
| US-002 Registration | C2 User Service, C5 HTTP | POST /api/v1/users | User | components, api-spec | ✅ Covered |
| US-003 Login → JWT | C2, C4, C8 JWT, C9 bcrypt | POST /api/v1/auth/login | User | components, api-spec | ✅ Covered |
| US-004 JWT middleware | C5 HTTP, C8 JWT | protected endpoints | — | components, api-spec | ✅ Covered |
| US-005 Get/List | C2, C5, C7 Mongo | GET /api/v1/users, GET /api/v1/users/:id | User | components, api-spec | ✅ Covered |
| US-006 Update/Delete | C2, C5, C7 | PUT/DELETE /api/v1/users/:id | User | components, api-spec | ✅ Covered |
| US-007 MongoDB persistence | C7 Mongo, C9 bcrypt | — | User | components, data-model, integration | ✅ Covered |
| US-008 Logging middleware | C10 Logging | all HTTP | — | components | ✅ Covered |
| US-009 Concurrency task | C11 Count loop, C7 | — | — | components, integration, nfr | ✅ Covered |
| US-010 Unit testing | C2, C5..C9, C12 | — | — | testing-strategy, implementation | ✅ Covered |
| US-011 Containerization | C12, Docker | — | — | implementation, nfr | ✅ Covered |
| US-012 Interface abstraction | C1, C2, C7 | — | — | components, integration | ✅ Covered |
| US-013 Input validation | C2, C5 | POST/PUT | User | components, api-spec | ✅ Covered |
| US-014 Graceful shutdown | C12, C11 | — | — | components, nfr | ✅ Covered |
| US-015 gRPC support | C6 gRPC | CreateUser/GetUser | User | components, api-spec, integration | ✅ Covered |
| US-016 Hexagonal architecture | C1..C12 | — | — | components, implementation | ✅ Covered |

### Coverage Summary
- **Requirements covered**: 16 / 16 จาก requirements.md
- **Gaps**: none
- **Components without requirement**: C10 Logging middleware maps ถึง US-008; C12 Composition root maps ถึง US-014/016; C3/C4 ports เป็น infrastructure/scaffolding ที่รองรับ US-003/012

## Open Questions & Risks

| # | Question/Risk | Impact | Status |
|---|--------------|--------|--------|
| 1 | `EstimatedDocumentCount` อาจ under-count หลังมีการลบ user | Low | Mitigated — เป็น job logging; ถ้าต้อง precision 100% ใช้ `CountDocuments` + hint `_id_` |
| 2 | recovery_middleware.go ไม่ถูกใช้ (พึ่ง Fiber built-in recover) | Low | Open — dead code; cleanup ได้ในงาน refactor |
| 3 | JWT ไม่มี refresh/revocation strategy | Low | Mitigated — deliverable non-production; เก็บเป็น future |

## Detailed Specifications

- [Components](design/components.md) — component breakdown และ interfaces
- [Data Model](design/data-model.md) — entities, schemas, indexes
- [API Specification](design/api-spec.md) — endpoints และ contracts (REST + gRPC)
- [Integration](design/integration.md) — external services และ integration testing
- [Implementation](design/implementation.md) — directory structure, setup, conventions
- [Testing Strategy](design/testing-strategy.md) — test architecture และ coverage mapping
- [Non-Functional Requirements](design/nfr.md) — performance, security, infrastructure

## External References

| Source | Type | Used in |
|--------|------|---------|
| `documents/requirements/user-management-api/requirement.md` | Requirement spec | ทั้งหมด |
| `documents/requirements/user-management-api/requirement_brd.md` | Implementation BRD | components, integration, nfr |
| `documents/requirements/user-management-api/tech-stack.md` | Tech stack/architecture doc | components, implementation |
| `api/openapi.yaml` | OpenAPI spec (source of truth) | api-spec |
| `api/proto/user_service.proto` | gRPC proto spec | api-spec |
| `go.mod` (module root) | Dependency manifest | implementation |
| MongoDB Go Driver — Count Documents docs | Mongodb.com documentation | data-model, nfr |