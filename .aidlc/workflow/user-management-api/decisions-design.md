# Design Decisions (D3)

## Context Summary

- **Type**: Brownfield · **Scope**: new · **Complexity**: Medium · **Team**: Solo (1 developer)
- **Stack**: Go 1.26.4 / Fiber v2 / MongoDB (mongo-driver v2) / JWT (HMAC HS256) / gRPC / bcrypt
- **Architecture**: Hexagonal (Ports & Adapters) — `adapter → application → domain`
- **Stories**: 16 (US-001..US-016) — 10 High / 4 Medium / 2 Low, 1 domain (user management)
- **Feature**: User Management API — register/login (JWT HS256), CRUD users, logging middleware, background count task ทุก 10 วิ, unit testing, bonus (Docker, validation, graceful shutdown, gRPC, hexagonal)
- **Requirements source**: `.aidlc/specs/user-management-api/requirements.md` + `documents/requirements/user-management-api/*`

---

## Decision Questions

### D3-1: Backend Framework
**Question**: ใช้ HTTP framework ตัวใดสำหรับ REST API?
- 1) Fiber v2 (ตาม stack เดิม; เร็ว, middleware-friendly, เข้ากับ oapi-codegen ได้) **(Recommended)**
- 2) Gin (popular, ใกล้เคียงกัน)
- 3) Echo (โครงสร้าง clean, built-in middleware)
- 4) stdlib `net/http` (ไม่พึ่ง framework)
- 5) Other (please specify): _______

**Answer**: 1) Fiber v2 (ตาม stack เดิม; เร็ว, middleware-friendly, เข้ากับ oapi-codegen ได้)

---

### D3-2: API Design Pattern
**Question**: รูปแบบ API ที่ต้องการ?
- 1) Hybrid — REST (Fiber + OpenAPI) และ gRPC (`CreateUser`/`GetUser`) **(Recommended)**
- 2) REST เท่านั้น
- 3) gRPC เท่านั้น
- 4) Other (please specify): _______

**Answer**: 1) Hybrid — REST (Fiber + OpenAPI) และ gRPC (`CreateUser`/`GetUser`)

---

### D3-3: API Documentation
**Question**: ใช้วิธีใดในการ define/document API?
- 1) OpenAPI 3.0.3 เป็น source of truth + oapi-codegen gen server interface (ตาม stack เดิม) **(Recommended)**
- 2) เขียน API docs ด้วยมือ
- 3) ไม่มี docs อย่างเป็นทางการ
- 4) Other (please specify): _______

**Answer**: 1) OpenAPI 3.0.3 เป็น source of truth + oapi-codegen gen server interface

---

### D3-4: Input Validation
**Question**: ใช้วิธีใดในการ validate input (required fields, email format)?
- 1) Custom validation ใน application layer (`validEmail`, `normalizeEmail`, password ≥ 8, body-size limit) **(Recommended)**
- 2) ใช้ library เช่น `go-playground/validator` (struct tags)
- 3) ทำทั้งสอง (custom + library)
- 4) Other (please specify): _______

**Answer**: 1) Custom validation ใน application layer (`validEmail`, `normalizeEmail`, password ≥ 8, body-size limit)

---

### D3-5: Error Handling Approach
**Question**: รูปแบบการจัดการ error เป็นอย่างไร?
- 1) Domain sentinel errors (`ErrNotFound`, `ErrDuplicateEmail`, `ErrInvalidCredentials`) + `ValidationError` → map เป็น HTTP/gRPC codes ที่ adapter layer **(Recommended)**
- 2) HTTP error middleware ใช้ status codes ตรง ๆ
- 3) Result types (`Result[T]`) คืนตลอด flow
- 4) Other (please specify): _______

**Answer**: 1) Domain sentinel errors + `ValidationError` → map เป็น HTTP/gRPC codes ที่ adapter layer

---

### D3-6: Logging Framework
**Question**: ใช้ logging framework ตัวใด?
- 1) Go stdlib `log/slog` (structured key-value logging, เข้ากับ stdlib, ใช้อยู่แล้วในโปรเจกต์) **(Recommended)**
- 2) `zerolog`
- 3) `zap`
- 4) Other (please specify): _______

**Answer**: 1) Go stdlib `log/slog`

---

### D3-7: Data Access Client (MongoDB)
**Question**: ใช้ client/library ตัวใดสำหรับ MongoDB?
- 1) Official `go.mongodb.org/mongo-driver/v2` (ตาม requirement ข้อ 4 + stack เดิม) **(Recommended)**
- 2) ODM/library ที่ mapper ให้เป็น struct อัตโนมัติ
- 3) ใช้ driver เจ้าเดียวกับ v1 (mongo-driver v1)
- 4) Other (please specify): _______

**Answer**: 1) Official `go.mongodb.org/mongo-driver/v2`

---

### D3-8: Caching Strategy
**Question**: ต้องการ caching ระดับใดสำหรับระบบนี้?
- 1) ไม่ใช้ caching ใน MVP — ข้อมูล users เล็ก, จุดเน้นคือ correctness **(Recommended)**
- 2) in-memory cache (เช่น sync.Map) สำหรับ List/Get
- 3) Redis cache
- 4) Other (please specify): _______

**Answer**: 1) ไม่ใช้ caching ใน MVP

---

### D3-9: Authentication Method
**Question**: ใช้วิธี authentication แบบใด?
- 1) JWT (HMAC HS256, secret key จาก config) + Bearer token middleware — ตาม requirement **(Recommended)**
- 2) OAuth 2.0 / OIDC
- 3) Session-based
- 4) Other (please specify): _______

**Answer**: 1) JWT (HMAC HS256) + Bearer token middleware

---

### D3-10: Password Hashing
**Question**: ใช้ algorithm ใดในการ hash password?
- 1) `bcrypt` (ใช้งานง่าย, cost ปรับได้, เหมาะกับ challenge นี้; ตาม stack เดิม) **(Recommended)**
- 2) `Argon2id` (ทันสมัยกว่า, memory-hard)
- 3) `scrypt`
- 4) Other (please specify): _______

**Answer**: 1) `bcrypt`

---

### D3-11: Integration Testing Approach
**Question**: ใช้วิธีใดทดสอบ integration กับ MongoDB/gRPC?
- 1) Opt-in live MongoDB test (`TestConnect_Live` ผ่าน `TEST_MONGO_URI`) + bufconn in-process สำหรับ gRPC + mock `mongo.Database` -- logistic paths **(Recommended)**
- 2) Testcontainers (spin up real MongoDB ใน container)
- 3) ทำทั้งสอง
- 4) Other (please specify): _______

**Answer**: 1) Opt-in live MongoDB test + bufconn gRPC + mock `mongo.Database`

---

### D3-12: Correctness & Property-Based Testing
**Question**: จะตรวจสอบ correctness ของระบบอย่างไร?
- 1) Table-driven tests + edge cases (duplicate email, invalid format, not-found, unauthorized) ด้วย `go test` stdlib — ไม่ใช้ PBT **(Recommended)**
- 2) เพิ่ม property-based testing (เช่น `rapid` / `gopter`) สำหรับ utils (normalizeEmail, validEmail)
- 3) ทำ table-driven และ PBT ทั้งสอง
- 4) Other (please specify): _______

**Answer**: 1) Table-driven tests + edge cases ด้วย `go test` — ไม่ใช้ PBT

---

### D3-13: Monitoring & Observability
**Question**: ระดับการ monitoring ที่ต้องการ?
- 1) Structured logging (slog) + logging middleware (method/path/status/duration) + background count log — ไม่มี metrics endpoint **(Recommended)**
- 2) เพิ่ม metrics endpoint (Prometheus `/metrics`)
- 3) Full observability (metrics + structured traces)
- 4) Other (please specify): _______

**Answer**: 1) Structured logging (slog) + logging middleware + background count log

---

### D3-14: Configuration & Secret Management
**Question**: จัดการ config/secret อย่างไร?
- 1) YAML config + env override + secret files per environment (ตาม stack เดิม: `config.Load()`, `APP_ENV`) **(Recommended)**
- 2) Environment variables เท่านั้น
- 3) External secrets manager (Vault / AWS Secrets Manager)
- 4) Other (please specify): _______

**Answer**: 1) YAML config + env override + secret files per environment

---

## Decisions Summary
<!-- Machine-readable compact summary. Downstream phases: read ONLY this section. -->
<!-- Auto-populated after user fills answers. One line per decision. -->
- D3-1 Backend framework: Fiber v2 (1)
- D3-2 API design: Hybrid REST + gRPC (1)
- D3-3 API documentation: OpenAPI 3.0.3 + oapi-codegen (1)
- D3-4 Validation: Custom in application layer (1)
- D3-5 Error handling: Sentinel errors + adapter mapping (1)
- D3-6 Logging: log/slog (1)
- D3-7 Data access: mongo-driver v2 official (1)
- D3-8 Caching: None for MVP (1)
- D3-9 Authentication: JWT HMAC HS256 + Bearer middleware (1)
- D3-10 Password hashing: bcrypt (1)
- D3-11 Integration testing: opt-in live + bufconn + mock mongo (1)
- D3-12 Correctness & PBT: Table-driven only, no PBT (1)
- D3-13 Monitoring: Structured logging only (1)
- D3-14 Config & secrets: YAML + env override + secret files (1)

---

**Instructions**: Fill in your answers above and respond with "done"