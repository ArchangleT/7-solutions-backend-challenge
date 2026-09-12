# Integration — User Management API

## Overview

ระบบนี้มี external integration หนึ่งตัว: **MongoDB** (persistence) และ integration "internal" อีก 2: **JWT signing** (in-process, secret จาก config) และ **gRPC inbound** (channel ขนานกับ REST — ไม่ใช่ external service). ไม่มี inter-unit communication (single unit — ไม่มีการเรียกข้าม service)

## External Integrations

### MongoDB (persistence)

| Property | Value |
|----------|-------|
| Purpose | เก็บผลลัพธ์ users + collection stats |
| Type | DB driver (official Go client) |
| Auth method | ตาม MongoDB URI (`MONGO_URI` / config) |
| Key operations | `InsertOne`, `FindOne`, `Find`, `UpdateOne`/`ReplaceOne`, `DeleteOne`, `EstimatedDocumentCount`, `CreateIndexes` |
| Error handling | wrap ด้วย `%w` (socket/network/server errors) → map เป็น 500 ที่ adapter layer; no-document → `domain.ErrNotFound`; duplicate key (unique email) → `ErrDuplicateEmail` |

**Retry/timeout**: connect ที่ startup พร้อม timeout; runtime errors ตกไปที่ `500 Internal Server Error` — ไม่มี silent failure (ทุก error ถูก wrap + log)

**Unique index**: `email_1` unique index สร้างตอน startup (`EnsureIndexes`) — เป็น defense กัน race duplicate email

### JWT (token signing — in-process)

| Property | Value |
|----------|-------|
| Purpose | sign/verify HS256 token |
| Type | library (`golang-jwt/jwt/v5`) — ไม่ใช่ external service |
| Auth method | HMAC secret key จาก config (`JWTSecret`) — ผ่าน `ports.TokenManager` |
| Key operations | `Generate(userID)`, `Parse(token)` |
| Error handling | parse fail → 401 (middleware) |

### gRPC (inbound channel — internal)

Internal inbound adapter; listen ที่ `GRPC_PORT` (default 50051) พร้อม `GracefulStop()`; error mapping → gRPC codes (ดู `design/api-spec.md`)

## Inter-Unit Communication

ไม่適用 — single unit workflow (ไมมี decomposition) ไม่มี cross-unit call

## Integration Testing

| Integration | Strategy | File |
|-------------|----------|------|
| MongoDB | mock `mongo.Database` และ collection interface (logic paths) + opt-in live test (`TestConnect_Live` ผ่าน env `TEST_MONGO_URI`) | `internal/adapter/outbound/mongodb/user_repository_test.go`, `user_repository_integration_test.go`, `internal/infrastructure/db/mongodb_test.go` |
| gRPC | bufconn in-process + fake `user.Service` (บังคับ auth token ที่ `GetUser`) | `internal/adapter/inbound/grpc/server_test.go` |
| Contract | OpenAPI spec เป็น source of truth (`api/openapi.yaml`) — routes gen ผ่าน oapi-codegen; proto def เป็น source for gRPC | `make codegen` / `make proto` |