# API Specification — User Management API

## Overview

- **API style**: Hybrid — REST (HTTP/Fiber + OpenAPI 3.0.3) + gRPC (`CreateUser`/`GetUser`)
- **Base URL**: `http://localhost:{PORT}` (HTTP, default 8080) · gRPC listen `:{GRPC_PORT}` (default 50051)
- **Auth method**: JWT (HMAC HS256) — `Authorization: Bearer <token>`
- **Content-Type**: `application/json`
- **API versioning**: URL-based `/api/v1`

## Conventions

- **Pagination**: none (รายการ users ขนาดเล็ก — คืนทั้งหมด)
- **Filtering/Sorting**: none
- **Rate limiting**: none (MVP — future: per-IP)
- **Versioning**: `/api/v1` ใน path
- **Validation**: 400 Bad Request ด้วยข้อความระบุ field

## Error Format

Error คืนเป็น envelope โดยมี `error` = ข้อความอ่านเข้าใจ พร้อม `fields` เฉพาะกรณี validation:

```json
{
  "error": "user not found"
}
```

Validation error variant (ผ่าน `WriteServiceError`):

```json
{
  "error": "validation error: email is invalid",
  "fields": {
    "email": "is invalid"
  }
}
```

**REST status mapping** (จาก domain/application error):

| Error | HTTP Status |
|-------|-------------|
| `domain.ErrNotFound` | 404 Not Found |
| `domain.ErrInvalidID` | 400 Bad Request |
| `domain.ErrDuplicateEmail` | 409 Conflict |
| `domain.ValidationError` | 400 Bad Request |
| `domain.ErrInvalidCredentials` | 401 Unauthorized |
| unknown / internal | 500 Internal Server Error |
| rate limited (`POST /api/v1/auth/login`) | 429 Too Many Requests |

**Authorization**:
- 401 Unauthorized เมื่อ missing/invalid Bearer token (จาก `middleware.Authenticate`)
- 403 Forbidden เมื่อ token ใช้ได้แต่ id ที่ขอไม่ใช่ subject ของ token (ownership check — GET/PUT/DELETE `/api/v1/users/:id` จำกัดเฉพาะเจ้าของบัญชี)

## REST Endpoints

### Public

#### `POST /health`
- **Auth**: none
- **Description**: health check
- **Response**: `200 OK` — `{"status": "ok"}`

#### `POST /api/v1/users` — Register / Create user
- **Auth**: none
- **Description**: สร้าง user ใหม่ (== register)
- **Source**: US-001, US-002, US-013

**Request body**:
```json
{
  "name": "Somchai Jaidee",
  "email": "somchai@example.com",
  "password": "secretPass123"
}
```

**Response** `201 Created`:
```json
{
  "id": "673adb1f2f4a9c8e7a1b2c3d",
  "name": "Somchai Jaidee",
  "email": "somchai@example.com",
  "created_at": "2026-09-13T01:00:00Z"
}
```
(ไม่มี `password` ใน response)

**Errors**: `400` validation (name required, email invalid, password < 8), `409` duplicate email, `500` internal

#### `POST /api/v1/auth/login` — Login
- **Auth**: none
- **Description**: verify credentials → return JWT
- **Source**: US-003, US-004

**Request body**:
```json
{
  "email": "somchai@example.com",
  "password": "secretPass123"
}
```

**Response** `200 OK`:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Errors**: `401` invalid credentials, `400` invalid input, `429` too many attempts (rate limited ต่อ IP — `middleware.RateLimit`, 10 ครั้ง/นาที) — normalที่ deploy หลัง reverse proxy/LB ต้องเปิด `server.trust_proxy: true` + ระบุ `server.trusted_proxies` เพื่อให้ Fiber อ่าน client IP จริง (ไม่งั้นทุก client ชน bucket เดียว)

### Protected (require `Authorization: Bearer <token>`)

#### `GET /api/v1/users` — List users
- **Auth**: required
- **Description**: คืนรายการ users ทั้งหมด
- **Source**: US-005

**Response** `200 OK`:
```json
[
  {
    "id": "673adb1f2f4a9c8e7a1b2c3d",
    "name": "Somchai Jaidee",
    "email": "somchai@example.com",
    "created_at": "2026-09-13T01:00:00Z"
  }
]
```

**Errors**: `401` unauthorized

#### `GET /api/v1/users/:id` — Get user by ID
- **Auth**: required
- **Description**: คืน user ตาม ID
- **Source**: US-005

**Response** `200 OK`:
```json
{
  "id": "673adb1f2f4a9c8e7a1b2c3d",
  "name": "Somchai Jaidee",
  "email": "somchai@example.com",
  "created_at": "2026-09-13T01:00:00Z"
}
```

**Errors**: `401`, `404` not found, `400` invalid ID

#### `PUT /api/v1/users/:id` — Update user
- **Auth**: required
- **Description**: อัปเดต name/email (email ตรวจ unique ซ้ำ)
- **Source**: US-006

**Request body**:
```json
{
  "name": "Somchai Updated",
  "email": "new@example.com"
}
```

**Response** `200 OK`:
```json
{
  "id": "673adb1f2f4a9c8e7a1b2c3d",
  "name": "Somchai Updated",
  "email": "new@example.com",
  "created_at": "2026-09-13T01:00:00Z"
}
```

**Errors**: `401`, `404`, `400` (invalid/empty fields), `409` duplicate email

#### `DELETE /api/v1/users/:id` — Delete user
- **Auth**: required
- **Description**: ลบ user
- **Source**: US-006

**Response** `200 OK` / `204 No Content` (ตาม OpenAPI)

**Errors**: `401`, `404`, `400`

## gRPC Definitions

Proto: `api/proto/user_service.proto` (gen stubs ผ่าน `make proto` → `internal/adapter/inbound/grpc/userv1/`)

```protobuf
syntax = "proto3";
package userv1;

service UserService {
  rpc CreateUser (CreateUserRequest) returns (User);  // public
  rpc GetUser (GetUserRequest) returns (User);        // requires Bearer token + subject == id
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  string password = 3;
}
message GetUserRequest { string id = 1; }
message User {
  string id = 1;
  string name = 2;
  string email = 3;
  string created_at = 4;  // RFC 3339
}
```

**Auth**: `GetUser` ต้องส่ง `authorization: Bearer <jwt>` ใน metadata (ตรวจด้วย unary interceptor) — `CreateUser` public. **Ownership**: subject ของ token ต้องตรงกับ `{id}` ที่ขอ มิฉะนั้นได้ `codes.PermissionDenied`

**gRPC error mapping**:

| Domain Error | gRPC Code |
|--------------|-----------|
| `ErrNotFound` | `codes.NotFound` |
| `ErrDuplicateEmail` | `codes.AlreadyExists` |
| `ErrInvalidID` / `ValidationError` | `codes.InvalidArgument` |
| `ErrInvalidCredentials` | `codes.Unauthenticated` |
| ownership mismatch | `codes.PermissionDenied` |
| other | `codes.Internal` (ข้อความ generic — ไม่ leak รายละเอียดภายใน) |

## Source Traceability

| Endpoint | Source Story |
|----------|-------------|
| POST /api/v1/users | US-001, US-002, US-013 |
| POST /api/v1/auth/login | US-003 |
| All protected endpoints | US-004 |
| GET /api/v1/users, GET /api/v1/users/:id | US-005 |
| PUT/DELETE /api/v1/users/:id | US-006 |
| GET /health | US-014 (health check) |
| gRPC CreateUser/GetUser | US-015 |