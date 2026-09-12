# Tasks — User Management API

**Scope**: new (comprehensive) · **Strategy**: Feature/module breakdown · **Approach**: Test-After · **Derived from**: design.md + design/*

## Overview

Brownfield — โค้ด Go implement ครบแล้วใน repo. Tasks เหล่านี้คือ bridge ระหว่าง design ที่ approved กับโค้ด: แต่ละ task = verify component กับ design spec → แก้/เพิ่มของที่ตามหลัง design → ยืนยันด้วย tests ตาม testing-strategy.md.

**Checkbox legend**: `- [ ] N.` = Phase · `  - [ ] N.M` = Task · bullets = details

---

## Task Phases

### 1. Infrastructure & Config Foundation

  - [x] 1.1 Verify Config loading (env + secret files + YAML)
      - **Deps**: — | **Ref**: design/components.md — C12 · design/implementation.md — Development Setup
        - ตรวจ `config/config.go`: `Load()` อ่าน YAML/env/secrets ตาม D3-14; `APP_ENV`, `PORT`, `GRPC_PORT`, `TEST_MONGO_URI`
        - เติม unit tests (table-driven, `config/config_test.go`) ถ้าขาด
      - 1) config/
      - ✅ 12/12 เป็น base

  - [x] 1.2 Verify MongoDB connect wrapper
      - **Deps**: — | **Ref**: design/components.md — C1·C7 · design/integration.md — MongoDB
      - `internal/infrastructure/db/mongodb.go`: `Connect` + timeout + `EnsureIndexes` (unique email)
      - tests: mock + opt-in live ตาม testing-strategy
      - ✅ ดูแล design/integration.md

### 2. Domain & Application Core

  - [x] 2.1 Verify Domain·User entity + sentinel errors
      - **Deps**: — | **Ref**: design/components.md — C1 · design/data-model.md — entities
      - `internal/domain/user/entity.go` + `errors.go` (ErrNotFound, ErrDuplicateEmail, ErrInvalidCredentials, ErrUser ตาม design)
      - unit tests entity/utils
      - ✅ US-001

  - [x] 2.2 Verify application ports (Hasher, Token)
      - **Deps**: 2.1 | **Ref**: design/components.md — C3·C4
      - `internal/application/ports/{hasher,token}.go` — interface ให้ outbound ใช้
      - ✅ US-012, US-016

  - [x] 2.3 Verify User Service (register / login / CRUD / validation)
      - **Deps**: 2.1, 2.2 | **Ref**: design/components.md — C2 · design/api-spec.md — REST endpoints
      - `internal/application/services/user/service.go`: ตรรกะ Register/Login/Get/List/Update/Delete, validation (email lowercase+format, password min 8), ใช้ ports เท่านั้น
      - unit tests (fake repo + gomock hasher/token, table-driven)
      - ✅ US-002, US-003, US-005, US-006, US-013

  - [x] 2.4 Verify StatusDB application service
      - **Deps**: 2.2 | **Ref**: design/components.md — C11
      - `internal/application/services/statusdb/service.go` — count logic ผ่าน `statusdb.Repository` (gomock test)
      - ✅ US-009

### 3. Outbound Adapters & Auth

  - [x] 3.1 Verify Mongo User Repository + unique index
      - **Deps**: 2.1, 2.3 | **Ref**: design/components.md — C7 · design/data-model.md — document_schema·indexes · design/integration.md — MongoDB
      - `internal/adapter/outbound/mongodb/user_repository.go`: CRUD + duplicate-key → ErrDuplicateEmail + no-doc → ErrNotFound; `EnsureIndexes`
      - unit (mock mongo) + opt-in live tests (testing-strategy)
      - ✅ US-005, US-006, US-007, US-012

  - [x] 3.2 Verify JWT manager (HMAC HS256)
      - **Deps**: 2.2 | **Ref**: design/components.md — C8 · design/nfr.md — Security
      - `internal/adapter/outbound/jwt/jwt.go`: Generate/Parse, parse fail → error → 401
      - unit tests (table-driven, tamper case)
      - ✅ US-003 (login), US-004

  - [x] 3.3 Verify bcrypt hasher
      - **Deps**: 2.2 | **Ref**: design/components.md — C9
      - `internal/adapter/outbound/bcrypt/bcrypt.go`: Hash/Compare; unit tests (correct + wrong password)
      - ✅ US-003

### 4. Inbound Adapters & Middleware

  - [x] 4.1 Verify HTTP routes/handlers + auth middleware + error mapping
      - **Deps**: 2.3, 3.1, 3.2 | **Ref**: design/components.md — C5 · design/api-spec.md — conventions·error_format
      - `internal/adapter/inbound/http/`: server + `user/handler.go` + auth middleware + helpers; map terrors: ErrNotFound→404, ErrDuplicateEmail→409, Validation→400, ErrInvalidCredentials→401, อื่น→500
      - handler tests (Fiber `app.Test`) ครบทุก endpoint
      - ✅ US-002..006, US-012

  - [x] 4.2 Verify gRPC server + auth interceptor
      - **Deps**: 2.3, 3.2 | **Ref**: design/components.md — C6 · design/api-spec.md — grpc_definitions
      - `api/proto/user_service.proto` + `internal/adapter/inbound/grpc/`: CreateUser (public) + GetUser (token from metadata → Unauthenticated)
      - bufconn tests
      - ✅ US-015

  - [x] 4.3 Verify logging middleware
      - **Deps**: — | **Ref**: design/components.md — C10 · design/nfr.md — Observability
      - `internal/adapter/middleware/log_middleware.go`: slog method/path/status/duration
      - ✅ US-008

### 5. Background & Composition

  - [ ] 5.1 Verify concurrency task (count loop)
      - **Deps**: 2.4, 3.1 | **Ref**: design/components.md — C11 · design/nfr.md — Availability
      - `cmd/api/main.go` countUsersLoop: ticker 10 วิ + slog log "total users"
      - ✅ US-009

  - [ ] 5.2 Verify composition root + graceful shutdown
      - **Deps**: 4.1, 4.2, 4.3, 5.1 | **Ref**: design/components.md — C12 · design/nfr.md — Availability·CI/CD
      - `cmd/api/main.go`: wiring ทั้งหมด, `signal.NotifyContext` → `app.Shutdown()` + `grpcSrv.GracefulStop()` + `client.Disconnect()` + หยุด loop
      - ✅ US-014, US-016

### 6. Integration Verification (Real MongoDB)

  - [ ] 6.1 Live MongoDB integration verify (opt-in)
      - **Deps**: 3.1, 5.2 | **Ref**: design/integration.md — Integration Testing · design/testing-strategy.md — Environment
      - รันด้วย `TEST_MONGO_URI`; `make test-integration`; ตรวจ CRUD + unique index + count
      - ✅ US-007, US-009

  - [ ] 6.2 gRPC integration verify (bufconn + live app wiring)
      - **Deps**: 4.2, 5.2 | **Ref**: design/integration.md · design/testing-strategy.md — per endpoint
      - bufconn ต่อ actual server; ตรวจ GetUser กับ/ไม่มี token
      - ✅ US-015

  - [ ] 6.3 Docker-compose + healthcheck verify
      - **Deps**: 5.2 | **Ref**: design/nfr.md — Infrastructure·Availability
      - `docker-compose up` → api + mongo:7 + healthcheck; `GET /health` ตอบ OK; ports 8080/50051
      - ✅ US-011, US-014(health)

### 7. Testing & Quality Gates

  - [ ] 7.1 Unit test sweep ครบทุก component
      - **Deps**: 1.1..5.2 | **Ref**: design/testing-strategy.md — coverage mapping
      - ตาม mapping: C1,C2(statusdb),C5,C6,C7,C8,C9,C10,C11,C12 — ทุก component ที่มีหน้าที่ต้องมี unit tests
      - ✅ US-010

  - [ ] 7.2 Full suite + coverage gate
      - **Deps**: 7.1, 6.1, 6.2 | **Ref**: design/testing-strategy.md — run_commands · design/nfr.md — CI/CD
      - `make test` ≥ 80% non-generated (exclude mocks/.gen.go/.pb.go); `make build` ผ่าน
      - ✅ US-010

  - [ ] 7.3 Traceability verification (US-001..US-016)
      - **Deps**: 7.2 | **Ref**: design.md — Traceability
      - ตรวจทุก US-* ว่ามี task + test ครบ; รายงานใน audit
      - ✅ US-016

---

## Task Summary

| ID | Task | Dependencies | Status |
|----|------|--------------|--------|
| 1.1 | Config loading verify | — | ✅ |
| 1.2 | MongoDB connect verify | — | ✅ |
| 2.1 | Domain·User entity + errors | — | ✅ |
| 2.2 | Application ports | 2.1 | ✅ |
| 2.3 | User Service | 2.1, 2.2 | ✅ |
| 2.4 | StatusDB service | 2.2 | ✅ |
| 3.1 | Mongo repo + index | 2.1, 2.3 | ✅ |
| 3.2 | JWT manager | 2.2 | ✅ |
| 3.3 | bcrypt hasher | 2.2 | ✅ |
| 4.1 | HTTP routes/handlers/auth/errors | 2.3, 3.1, 3.2 | ✅ |
| 4.2 | gRPC server + interceptor | 2.3, 3.2 | ✅ |
| 4.3 | Logging middleware | — | ✅ |
| 5.1 | Count loop | 2.4, 3.1 | ⬜ |
| 5.2 | Composition + graceful shutdown | 4.1, 4.2, 4.3, 5.1 | ⬜ |
| 6.1 | Live Mongo integration | 3.1, 5.2 | ⬜ |
| 6.2 | gRPC integration | 4.2, 5.2 | ⬜ |
| 6.3 | Docker-compose + health | 5.2 | ⬜ |
| 7.1 | Unit test sweep | 1.1..5.2 | ⬜ |
| 7.2 | Full suite + coverage gate | 7.1, 6.1, 6.2 | ⬜ |
| 7.3 | Traceability verification | 7.2 | ⬜ |

## Requirements Coverage

| US-ID | Description | Implementing tasks | Status |
|-------|-------------|--------------------|--------|
| US-001 | User Model | 2.1 | ✅ |
| US-002 | Registration | 2.3, 4.1 | ✅ |
| US-003 | Login → JWT | 2.3, 3.2, 3.3, 4.1 | ✅ |
| US-004 | JWT middleware | 4.1, 3.2 | ✅ |
| US-005 | Get/List | 2.3, 3.1, 4.1 | ✅ |
| US-006 | Update/Delete | 2.3, 3.1, 4.1 | ✅ |
| US-007 | MongoDB persistence | 3.1, 1.2, 6.1 | ✅ |
| US-008 | Logging middleware | 4.3 | ✅ |
| US-009 | Concurrency task | 5.1, 2.4, 6.1 | ✅ |
| US-010 | Unit testing | 7.1, 7.2 | ✅ |
| US-011 | Containerization | 6.3 | ✅ |
| US-012 | Interface abstraction | 2.2, 3.1, 4.1 | ✅ |
| US-013 | Input validation | 2.3, 4.1 | ✅ |
| US-014 | Graceful shutdown | 5.2, 6.3 | ✅ |
| US-015 | gRPC support | 4.2, 6.2 | ✅ |
| US-016 | Hexagonal architecture | 5.2, 2.2, 7.3 | ✅ |

## Design Coverage

| Component | Task IDs | Entity | Task IDs | Endpoint | Task IDs |
|-----------|----------|--------|----------|----------|----------|
| C1 Domain·User | 2.1 | User | 3.1 (repo) → via 2.3 | POST /health | 4.1, 6.3 |
| C2 User Service | 2.3 | | | POST /api/v1/users | 2.3, 4.1 |
| C3 Hasher port | 2.2 | | | POST /api/v1/auth/login | 2.3, 4.1 |
| C4 Token port | 2.2 | | | GET /api/v1/users | 2.3, 4.1 |
| C5 HTTP Adapter | 4.1 | | | GET /api/v1/users/:id | 2.3, 4.1 |
| C6 gRPC Adapter | 4.2 | | | PUT /api/v1/users/:id | 2.3, 4.1 |
| C7 Mongo Adapter | 3.1 | | | DELETE /api/v1/users/:id | 2.3, 4.1 |
| C8 JWT manager | 3.2 | | | gRPC CreateUser | 4.2, 6.2 |
| C9 bcrypt hasher | 3.3 | | | gRPC GetUser | 4.2, 6.2 |
| C10 Logging middleware | 4.3 | | | Integrations: MongoDB 3.1·6.1, JWT 3.2, gRPC 4.2·6.2 | |
| C11 Count loop | 2.4, 5.1 | | | | |
| C12 Composition root | 1.1, 5.2 | | | | |

## Testing Coverage

- **unit_test_tasks**: 2.1, 2.2, 2.3, 2.4, 3.1, 3.2, 3.3, 4.1, 4.2, 4.3, 5.1, 7.1 (ทุก component C1–C12)
- **integration_test_tasks**: 6.1 (Mongo live), 6.2 (gRPC bufconn), 6.3 (docker health), 7.2 (full gate)
- **e2e_test_tasks**: none (E2E framework ไม่ถูกเลือกใน D3)
- **load_test_tasks**: none (load testing ไม่ถูกเลือกใน D3)
- **pbt_tasks**: none (PBT no — D3-12, D4-5)
- **coverage_summary**: components 12/12 · endpoints 9/9 (7 REST + 2 gRPC) · entity 1/1

## Definition of Done

- [ ] ทุก task มี `**Ref**` ชี้ไปที่ design file + section
- [ ] โค้ดตรงตาม design (components, data-model, api-spec, integration)
- [ ] `make test` ผ่าน + coverage ≥ 80% non-generated (7.2)
- [ ] Integration (live mongo opt-in, bufconn, docker) ผ่าน (6.x)
- [ ] ทุก US-001..US-016 มี task + test ครบ (7.3)
- [ ] ไม่มี dead code / recovery_middleware cleanup note (open risk #2)

## Execution Waves

| Wave | Phases | Resolved dependencies | Parallel |
|------|--------|----------------------|----------|
| 1 | 1, 2 | — (skeleton + logic) | 🟢 (2 phases) |
| 2 | 3, 4 | 2 (service/ports); 4 deps 2 | 🟢 (2 phases) |
| 3 | 5 | 3, 4 (adapters ready) | ⚪ |
| 4 | 6 | 5 (runnable app) | ⚪ |
| 5 | 7 | 6 (integration done) | ⚪ |

### File Ownership (no overlap)

| Wave | Phase | Owned files/dirs |
|------|-------|------------------|
| 1 | 1 | config/ · internal/infrastructure/db/ |
| 1 | 2 | internal/domain/ · internal/application/ · internal/mocks/ |
| 2 | 3 | internal/adapter/outbound/ |
| 2 | 4 | internal/adapter/inbound/ · internal/adapter/middleware/ · api/openapi.yaml · api/proto/ |
| 3 | 5 | cmd/api/ (main.go) |
| 4 | 6 | Dockerfile · docker-compose.yml · Makefile (test-integration targets) |
| 5 | 7 | internal/**/*_test.go (test files only) |

*No overlap — phases 1+2, 3+4 operate on disjoint paths; wave 5 touches test files only.*

## Notes

### Technical debt
- `internal/adapter/middleware/recovery_middleware.go` ยังไม่ได้ใช้ (design risk #2) — งาน cleanup ได้ใน 5.2
- `EstimatedDocumentCount` under-count หลังลบ user (design risk #1) — ยอมรับสำหรับ MVP

### Future enhancements
- Refresh/revocation strategy สำหรับ JWT (design risk #3)
- กรณี scale: เพิ่ม instance ได้ (stateless API + shared MongoDB)