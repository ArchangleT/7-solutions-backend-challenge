# Testing Strategy — User Management API

## Overview

- **Testing philosophy**: Test-After (โค้ดถูกเขียนแล้วใน brownfield — ผสานกับ table-driven เพื่อครอบคลุม edge cases)
- **Coverage target**: ≥ 80% บน non-generated code (ยกเว้น `internal/mocks/`, `server_openapi.gen.go`, `*.pb.go`)
- **Test pyramid**: Unit-heavy (service/handler/adapter/utils) + integration opt-in (MongoDB live, bufconn gRPC) — ไม่มี E2E/UI (ไม่มี frontend)

## Frameworks

| Level | Framework | Notes |
|-------|-----------|-------|
| Unit | Go stdlib `testing` + table-driven + `go.uber.org/mock` (gomock) | D3-12: no PBT |
| Integration | mock `mongo.Database` collections (logic paths) + opt-in live `TestConnect_Live` (`TEST_MONGO_URI`) | D3-11 |
| gRPC | bufconn in-process + fake `user.Service` | D3-11 |
| E2E | none (backend-only API; handler ทดสอบใน-process ผ่าน Fiber `app.Test`) | — |
| PBT | skipped — D3-12 = table-driven only | — |
| Load | none (MVP) | — |
| API | ใน-process ผ่าน Fiber `app.Test` (ไม่ต้อง Postman) | — |

## Test Architecture

### Directory structure

```
$PACKAGE/_test.go                       # table-driven unit tests (co-located)
internal/mocks/                         # gomock: statusdb/repository.go, userrepo/repository.go, ports/{token,hasher}.go
internal/adapter/outbound/mongodb/
  user_repository_test.go               # logic paths (mock mongo database)
  user_repository_integration_test.go   # opt-in live (TEST_MONGO_URI)
internal/infrastructure/db/mongodb_test.go
```

### Naming conventions
- File: `{name}_test.go`; integration live test: `TestConnect_Live`
- Test cases: table-driven (`tests := []struct{ name string; ... }`)
- Mocks: ตั้งชื่อตาม interface (`MockRepository`, `MockHasher`, `MockTokenManager`)

### Shared utilities
- fake in-memory repository สำหรับ `user.Service` tests (ตัวจริงของ `errors.Is` API)
- `bindata`/fixtures: ไม่ใช้ — ข้อมูลใน test สร้าง inline
- gomock generator: `make mock` (mockgen จาก port interfaces)

### Test data management
- Unit: ไม่ต้อง DB — ใช้ fake/mock repo
- Integration live: ต่อ MongoDB จริง `TEST_MONGO_URI` (opt-in — ไม่รันใน CI ปกติ) — ใช้ database แบบชั่วคราว/cleanup ใน test
- Isolation: แต่ละ test สร้าง data เอง; ไม่แชร์ state

## Mock Strategy

| Dependency | Approach |
|------------|----------|
| MongoDB | mock `mongo.Database`/collection interface (`internal/adapter/outbound/mongodb/user_repository_test.go`) + opt-in live |
| `user.Repository` / `statusdb.Repository` | fake in-memory (user service) / gomock (statusdb service) |
| `ports.Hasher` / `ports.TokenManager` | gomock mocks ที่ `internal/mocks/ports/` |
| grpc client | bufconn (in-process `grpc.NewServer` + listener) |
| time | `time.NewTicker` ใน `countUsersLoop` — ไม่ mock (interval 10s); test ผ่าน unit ของ `statusdb.Service` ด้วย gomock repo |

## Environment

- **Test database**: ไม่ต้องใช้สำหรับ unit; integration ใช้ live MongoDB ผ่าน `TEST_MONGO_URI`
- **Env variables**: `TEST_MONGO_URI` (integration เท่านั้น)
- **Containers**: ไม่บังคับ (ไม่มี TestContainers) — รัน MongoDB ด้วย docker-compose/infra เดียวกัน
- **CI integration**: `make test` เป็นเกณฑ์หลัก; `make test-integration` แยก (ต้องมี MongoDB URI)

## Coverage Mapping

### Per component (อ้างอิง `design/components.md`)

| Component | Test type | File |
|-----------|-----------|------|
| C2 User Service | unit (table-driven, fake repo) | `internal/application/services/user/service_test.go`, `utils_test.go` |
| C11 StatusDB | unit (gomock repo) | `internal/application/services/statusdb/service_test.go` |
| C5 HTTP Adapter | handler (Fiber `app.Test`) | `internal/adapter/inbound/http/user/handler_test.go`, `health-check/handler_test.go`, `helpers/utils_test.go` |
| C6 gRPC Adapter | integration (bufconn) | `internal/adapter/inbound/grpc/server_test.go` |
| C7 Mongo Adapter | unit (mock mongo) + opt-in live | `mongodb/user_repository_test.go`, `user_repository_integration_test.go` |
| C8 JWT | unit (table-driven) | `internal/adapter/outbound/jwt/jwt_test.go` |
| C9 bcrypt | unit (table-driven) | `internal/adapter/outbound/bcrypt/bcrypt_test.go` |
| C12 Config | unit (table-driven) | `config/config_test.go` |
| Composition (shutdown) | manual/covered via handler tests | — |

### Per endpoint (อ้างอิง `design/api-spec.md`)

| Endpoint | Test type |
|----------|-----------|
| POST /api/v1/users | handler test (201, 400 validation, 409 duplicate) |
| POST /api/v1/auth/login | handler test (200 + token, 401) |
| GET /api/v1/users (+/:id) | handler test (200, 404, 401) |
| PUT/DELETE /api/v1/users/:id | handler test (200, 400, 404, 409, 401) |
| GET /health | handler test (table-driven) |
| gRPC CreateUser/GetUser | bufconn test (success + unauth) |

### Critical paths
- Register → duplicate email → 409 (validation + race prevention สองชั้น: app check + unique index)
- Login ผิด/ถูก credentials
- Guarded endpoint ไม่มี/มี token

## Run Commands

| Command | Description |
|---------|-------------|
| `make test` | full suite + coverage (exclude generated/mocks) |
| `go test ./...` | unit tests ทั้งหมด |
| `make test-integration` | opt-in live MongoDB (ต้อง `TEST_MONGO_URI`) |
| `go test ./internal/adapter/inbound/grpc/...` | gRPC bufconn tests |
| `go test -coverprofile=coverage.out ./...` | coverage profile |