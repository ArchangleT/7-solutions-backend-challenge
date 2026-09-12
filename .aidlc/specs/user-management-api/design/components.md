# Components — User Management API

## Overview

ระบบเป็น monolith เดียว (single binary `bin/api_user_mgr`) ที่ใช้ **Hexagonal Architecture (Ports & Adapters)** ประกอบด้วย inbound adapters (HTTP/Fiber + gRPC) เรียกใช้ **application services** (use cases) ซึ่งพึ่งพา **domain ports** (interfaces) และถูก implement โดย outbound adapters (mongodb / jwt / bcrypt)

Dependency direction: `adapter → application → domain ← adapter (outbound)`. Domain ไม่รู้จัก Fiber/HTTP/MongoDB/gRPC

```
HTTP client ──► Fiber (oapi-codegen) ──► middleware.Authenticate ──► user.Handler ──┐
gRPC client ──► grpc.Server + tokenAuth ─► userv1.UserServiceServer ────────────┤
                                                                               ▼
                                                             application/services/user.Service
                                                                               │
Dimension: JWT (login/register)  bcrypt (hash)                domain/user.Repository (port)
                                                                               ▼
                                                        outbound/mongodb.UserRepository
                                                                               │
                                                        MongoDB (users)

Background: countUsersLoop (10s ticker) → statusdb.Service → domain/statusdb.Repository → mongodb.Count (EstimatedDocumentCount)
```

## Components

### C1: Domain — User
- **Purpose**: Business core ของระบบ — entity + business rules สำหรับ user
- **Technology**: Go (no external deps)
- **Responsibilities**: นิยาม `User` entity ที่มีฟิลด์ `ID, Name, Email, Password (hashed), CreatedAt`; ตรวจ email format, normalize lowercase, password policy (min 8)
- **Exposes**: `User` struct, `Validate()`, sentinel errors (`ErrNotFound`, `ErrDuplicateEmail`, `ErrInvalidCredentials`), `ValidationError`
- **Consumes**: —
- **Internal structure**: `internal/domain/user/` (`entity.go`, `errors.go`, `repository.go`) + `internal/domain/statusdb/repository.go` (port แยกสำหรับ count)
- **Key decisions**:
  1. แยก `statusdb.Repository` ออกจาก `user.Repository` เพื่อไม่ให้ user service ต้องรับภาระ collection stats
  2. เก็บ business rules (email normalize/validate, password length) ไว้ใน domain/app layer — ไม่ใช่ handler
- **Error handling**: คืน sentinel errors ให้ application layer ตัดสินใจ map เป็น HTTP/gRPC codes
- **Maps to stories**: US-001, US-012, US-016

### C2: Application — User Service
- **Purpose**: Orchestration/use cases ของทุก user flow
- **Technology**: Go (stdlib only + domain ports)
- **Responsibilities**: `Register`, `Login`, `List`, `GetUser`, `UpdateUser`, `DeleteUser`, `Count`; ทำงานผ่าน ports (`user.Repository`, `ports.Hasher`, `ports.TokenManager`)
- **Exposes**: `Service` interface + DTOs (`CreateUserInput`, `LoginInput`, `UserOutput`, `UpdateUserInput`)
- **Consumes**: `domain/user.Repository`, `domain/statusdb.Repository`, `application/ports.Hasher`, `application/ports.TokenManager`
- **Internal structure**: `internal/application/services/user/` (`service.go`, `dto.go`, `utils.go`) + `internal/application/services/statusdb/service.go`
- **Key decisions**:
  1. Register flow: validate → ตรวจ duplicate email → hash password → save → return output (โดยไม่ return password)
  2. Login flow: `FindByEmail` → `hasher.Compare` → `tokens.Generate(id)`
  3. `normalizeEmail` ทำ lowercase ก่อนทุก operation
- **Error handling**: `ValidationError` สำหรับ input ผิด; ส่งต่อ domain sentinel errors
- **Maps to stories**: US-002, US-003, US-005, US-006, US-007, US-009

### C3: Application — Password Hasher Port
- **Purpose**: Port abstraction สำหรับ password hashing
- **Technology**: Go interface (`ports.Hasher`)
- **Responsibilities**: `Hash(password)`, `Compare(hashed, plain)`
- **Consumes**: —
- **Key decisions**: Domain/application ไม่รู้ implementation — สลับ bcrypt/argon2 ได้โดยไม่กระทบ core
- **Maps to stories**: US-003, US-012

### C4: Application — Token Manager Port
- **Purpose**: Port abstraction สำหรับ JWT sign/parse
- **Technology**: Go interface (`ports.TokenManager`)
- **Responsibilities**: `Generate(id)`, `Parse(token)` — ใช้ร่วมกันโดย HTTP middleware และ gRPC auth
- **Key decisions**: Adapter ใช้ `golang-jwt/jwt/v5` HMAC HS256; PHP test ผ่าน mock (gomock)
- **Maps to stories**: US-003, US-004, US-015

### C5: Inbound — HTTP Adapter (Fiber + oapi-codegen)
- **Purpose**: เผย REST endpoints
- **Technology**: Fiber v2 + `server_openapi.gen.go` (oapi-codegen)
- **Responsibilities**: จัดการ routing (`GET /health`, `POST /api/v1/users`, `POST /api/v1/auth/login`, `GET /api/v1/users`, `GET/PUT/DELETE /api/v1/users/:id`); decode/encode body; map service error → HTTP status
- **Exposes**: HTTP endpoints (ดู `design/api-spec.md`)
- **Consumes**: `application/services/user.Service`, `ports.TokenManager`
- **Internal structure**: `internal/adapter/inbound/http/` (`server.go`, `server_openapi.gen.go`, `user/handler.go`, `health-check/`, `helpers/utils.go`) + `internal/adapter/middleware/`
- **Key decisions**: route ต่อ middleware `Authenticate` (Bearer JWT) สำหรับ guarded ops; `Responder` interface ทำ `DecodeBody` + `WriteServiceError` ร่วมกัน; body-size limit
- **Error handling**: map `ErrNotFound`→404, `ErrDuplicateEmail`→409, `ErrInvalidID`/`ValidationError`→400, `ErrInvalidCredentials`→401, อื่น ๆ→500
- **Maps to stories**: US-002, US-003, US-004, US-005, US-006, US-008, US-013

### C6: Inbound — gRPC Adapter
- **Purpose**: เผย `CreateUser`/`GetUser` ผ่าน gRPC
- **Technology**: `google.golang.org/grpc` + protoc-generated stubs (`userv1/*.pb.go`)
- **Responsibilities**: implement `userv1.UserServiceServer`; unary interceptor ตรวจ Bearer token ใน `authorization` metadata (เฉพาะ `GetUser`); map domain error → gRPC codes
- **Exposes**: `CreateUser` (public), `GetUser` (auth required)
- **Consumes**: `application/services/user.Service`, `ports.TokenManager`
- **Internal structure**: `internal/adapter/inbound/grpc/` (`server.go`, `auth.go`, `userv1/`)
- **Key decisions**: tokenAuth interceptor ใช้ tokenManager ร่วมกับ HTTP middleware; error mapping: NotFound→`codes.NotFound`, ErrDuplicateEmail→`AlreadyExists`, ErrInvalidID/Validation→`InvalidArgument`, ErrInvalidCredentials→`Unauthenticated`, อื่น ๆ→`Internal`; listen ที่ `GRPC_PORT` (default 50051) + `GracefulStop()`
- **Maps to stories**: US-015

### C7: Outbound — MongoDB Adapter
- **Purpose**: Implementation ของ `user.Repository` + `statusdb.Repository`
- **Technology**: `go.mongodb.org/mongo-driver/v2` (official)
- **Responsibilities**: CRUD user (`Create`, `FindByID`, `FindByEmail`, `List`, `Update`, `Delete`), count (`EstimatedDocumentCount`), สร้าง unique index บน email
- **Exposes**: implements domain ports
- **Consumes**: MongoDB (driver)
- **Internal structure**: `internal/adapter/outbound/mongodb/user_repository.go`
- **Key decisions**: ใช้ unique email index (สร้างผ่าน `EnsureIndexes`); count ใช้ `EstimatedDocumentCount` เพราะเป็น job สำหรับ logging (ความเร็วสำคัญกว่า precision หลักหน่วย) — ถ้าต้องการ precision 100% ใช้ `CountDocuments` + hint `_id_`
- **Error handling**: wrap error ด้วย `fmt.Errorf("...: %w")`; map no-document → `domain.ErrNotFound`
- **Maps to stories**: US-007, US-009, US-012

### C8: Outbound — JWT Manager
- **Purpose**: Implementation ของ `ports.TokenManager` (HMAC HS256)
- **Technology**: `golang-jwt/jwt/v5`
- **Responsibilities**: `Generate(userID)` → signed JWT; `Parse(token)` → claims
- **Consumes**: JWT secret (จาก config)
- **Key decisions**: sign ด้วย HS256 + secret key; แยกใน adapter เพื่อให้ test ใช้ mock ได้
- **Maps to stories**: US-003, US-004

### C9: Outbound — bcrypt Hasher
- **Purpose**: Implementation ของ `ports.Hasher`
- **Technology**: `golang.org/x/crypto/bcrypt`
- **Responsibilities**: `Hash`/`Compare` password
- **Key decisions**: cost default; แยกเป็น adapter ให้สลับ algorithm ได้
- **Maps to stories**: US-003, US-007

### C10: Middleware — Logging
- **Purpose**: logging middleware (method, path, status, duration)
- **Technology**: Fiber middleware + `log/slog`
- **Responsibilities**: log ทุก request
- **Maps to stories**: US-008, US-013

### C11: Application — Background Count Loop (statusdb)
- **Purpose**: concurrency task นับ total users ทุก 10 วินาที
- **Technology**: Go goroutine + `time.NewTicker` + `slog`
- **Responsibilities**: ทุก 10 วิ เรียก `statusdb.Service.Count` แล้ว log; หยุดเมื่อ `ctx.Done()` (graceful shutdown)
- **Consumes**: `statusdb.Service` → `domain/statusdb.Repository`
- **Key decisions**: ใช้ `time.NewTicker` (แม่นยำ) แทน `time.Sleep`; ปิด ticker ตอนจบ; error → log + continue (ไม่แครช)
- **Maps to stories**: US-009

### C12: Composition Root — main.go
- **Purpose**: DI ทั้งระบบ + lifecycle
- **Technology**: Go
- **Responsibilities**: โหลด config; connect MongoDB; ประกอบ userRepo/service/token/hasher; ขึ้น Fiber + gRPC; รัน count loop; รอ signal → shutdown gracefully (`app.Shutdown()`, `grpcSrv.GracefulStop()`, `client.Disconnect()`)
- **Consumes**: ทุก component
- **Key decisions**: เป็นจุดเดียวที่ infrastructure ถูกประกอบเข้ากับ application (ตาม hexagonal)
- **Maps to stories**: US-014, US-016

## Interactions

```
HTTP client ─► [C5 HTTP Adapter] ─► middleware.Authenticate ─► user.Handler ─► [C2 User Service]
gRPC client ─► [C6 gRPC Adapter] ─► tokenAuth interceptor ─► userv1.Server ─► [C2 User Service]
[C2 User Service] ─► [C7 Mongo Adapter] ─► MongoDB
[C2 User Service] ─► [C8 JWT]  /  [C9 bcrypt]
[countUsersLoop] ─► [C11 StatusDB] ─► [C7 Mongo Adapter].Count
```

**Data flow (Request lifecycle)**:
```
HTTP:  Fiber + oapi-codegen → middleware.Authenticate (JWT) → user.Handler (DecodeBody) → user.Service (validate → port) → mongodb repo → MongoDB → reverse mapping → JSON response (WriteServiceError on error)
gRPC:  grpc.Server → tokenAuth interceptor → userv1.UserServiceServer → user.Service → port → mongodb repo → MongoDB → proto response
```

**Cycle check**: `adapter → application → domain` และ `application → domain ports ← outbound adapter`. ไม่มี cycle