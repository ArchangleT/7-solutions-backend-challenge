# Data Model — User Management API

## Overview

- **Database**: MongoDB (document)
- **Client**: official `go.mongodb.org/mongo-driver/v2`
- **Database name**: จาก config (`MongoDatabase`, default `userdb` — ดู config)
- **Collection**: `users`

## Entities

### User (`users` collection)

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `_id` | `ObjectID` (driver auto) | ✅ | Primary key — auto-generated โดย MongoDB (mapped เป็น `User.ID` string) |
| `name` | `string` | ✅ | non-empty (trimmed) |
| `email` | `string` | ✅ | unique (lowercase, valid format) |
| `password` | `string` | ✅ | bcrypt-hashed — ไม่เคยเก็บ plaintext |
| `created_at` | `BSON DateTime` (UTC) | ✅ | set ตอนสร้าง (`time.Now().UTC()`) |

**Relationships**: —

**Indexes**:

| Name | Fields | Unique | Purpose |
|------|--------|--------|---------|
| `email_1` | `email` (asc) | ✅ | บังคับ unique email — สร้างผ่าน `EnsureIndexes` ตอน startup และเป็นกลไกกัน duplicate ที่ level database |

**Business rules** (ไม่สามารถ dump ลง schema ได้ทั้งหมด — อยู่ที่ application layer):
- email: normalize เป็น lowercase (`normalizeEmail`) ก่อนทุก operation; ตรวจ format (`validEmail`)
- password: อย่างน้อย 8 ตัวอักษร (ยืนยันที่ register)
- `Update`: เปลี่ยนได้เฉพาะ `name`/`email` — ตรวจ duplicate email ถ้ามีการเปลี่ยน email ใหม่
- `Delete`: hard delete

## Domain Entity vs Document Mapping

Domain `User` (internal/domain/user/entity.go) ↔ BSON document:

```
User{ ID, Name, Email, Password, CreatedAt }
        ↕  (bson tags)
users doc { _id, name, email, password, created_at }
```

- `ID` เป็น string แปลงจาก/to `ObjectID` ที่ repository layer
- การ persist/อ่านทั้งหมดผ่าน `internal/adapter/outbound/mongodb/user_repository.go`

## ER Diagram

```
┌──────────────────────────────┐
│          users               │
│──────────────────────────────│
│ _id        ObjectID (PK)     │
│ name       string            │
│ email      string  (unique*) │
│ password   string (hashed)   │
│ created_at datetime          │
└──────────────────────────────┘
   * unique index: email_1
```

ระบบมี 1 entity (single-collection design) — ไม่มีความสัมพันธ์/foreign keys

## Access Patterns

| Query | Frequency | Command | Index used |
|-------|-----------|---------|------------|
| Insert new user | ทุก register | `InsertOne` | — |
| Find user by email | ทุก register/login | `FindOne({email})` | `email_1` |
| Find user by ID | get/update/delete | `FindOne({_id})` | `_id_` |
| List all users | list | `Find({})` | collection scan (ข้อมูลน้อย) |
| Update user | update name/email | `ReplaceOne/UpdateOne` | `_id_` |
| Delete user | delete | `DeleteOne` | `_id_` |
| Count total users | background task ทุก 10 วิ | `EstimatedDocumentCount()` | metadata (no scan) |

## Document Schema (MongoDB)

ตัวอย่าง document `users`:

```json
{
  "_id": "ObjectId('673adb1f2f4a9c8e7a1b2c3d')",
  "name": "Somchai Jaidee",
  "email": "somchai@example.com",
  "password": "$2a$10$N9qo8uLOickgx2ZMRZoMye...",
  "created_at": "2026-09-13T01:00:00Z"
}
```

**Embedding rationale**: ไม่ต้องทำ embedding — user เป็น aggregate เดี่ยว ไม่มี sub-documents; ระบบนี้ใช้แค่ collection `users` เดียว

**Index note**: `email_1` เป็น unique index — เป็น last-line-of-defense กัน `ErrDuplicateEmail` (app layer ตรวจ `FindByEmail` ก่อน) แม้เกิด race window ระหว่าง register พร้อมกัน