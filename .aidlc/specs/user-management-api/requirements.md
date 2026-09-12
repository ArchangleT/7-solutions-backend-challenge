# Requirements — User Management API

## Summary
- **Total Stories**: 16 — ครอบคลุม 7 core requirements + 6 bonus (D1)
- **Functional Areas**: 5 — Authentication & User Model, User Operations, Persistence, Observability & Operations, Quality (+ Bonus)
- **Priority Distribution**: 10 High · 4 Medium · 2 Low
- **User Types**: API consumer / backend integrator (1)
- **Key Entities**: User (ID, Name, Email unique, Password hashed, CreatedAt)
- **Integrations**: MongoDB (persistence), JWT (HMAC HS256), bcrypt, gRPC (bonus)
- **Core Flows**:
  1. Register → hash password → save → 201
  2. Login → verify → คืน JWT
  3. เรียก guarded endpoint → ตรวจ Bearer token → CRUD
  4. Background task ทุก 10 วิ → log จำนวน users ทั้งหมด

## Functional Areas

### Area A: Authentication & User Model

#### US-001: User Model
- **As a** API consumer
- **I want** ระบบให้มี User entity ที่มีฟิลด์ครบตาม spec
- **So that** ข้อมูลผู้ใช้ถูกนิยามเป็นมาตรฐานเดียวกัน (ID, Name, Email unique, Password hashed, CreatedAt)
- **Priority**: High
- **Source**: D1-5, requirement.md §1

**Acceptance Criteria (EARS)**:
1. WHEN ระบบสร้าง User ใหม่ THEN ระบบ SHALL กำหนด `ID` (auto-generated), `Name`, `Email`, `Password` (hashed), `CreatedAt` ให้ครบ
2. WHERE User ถูกบันทึกลง persistence THEN `Email` SHALL เป็นค่า unique
3. WHERE User ถูกสร้าง THEN `Password` SHALL ถูกเก็บในรูปแบบ hashed (ไม่ใช่ plaintext)

#### US-002: Registration (สร้างผู้ใช้)
- **As a** API consumer
- **I want** register ผู้ใช้ใหม่ผ่าน `POST /api/v1/users` (public)
- **So that** ผู้ใช้ใหม่ถูกเก็บในระบบ พร้อมกลับ response success
- **Priority**: High
- **Source**: D1-4, requirement.md §2/3

**Acceptance Criteria (EARS)**:
1. WHEN ระบบได้รับ register request ที่ข้อมูลครบถ้วนและ valid THEN ระบบ SHALL hash password, บันทึก user ใหม่ และตอบ 201 พร้อมข้อมูลผู้ใช้ (ไม่รวม password)
2. WHEN email ซ้ำกับ user ที่มีอยู่ THEN ระบบ SHALL ปฏิเสธด้วย error duplicate email
3. WHEN password สั้นกว่า 8 ตัวอักษร THEN ระบบ SHALL ปฏิเสธด้วย validation error
4. WHEN email มีรูปแบบไม่ถูกต้อง THEN ระบบ SHALL ปฏิเสธด้วย validation error

#### US-003: Login คืน JWT
- **As a** API consumer
- **I want** เข้าสู่ระบบผ่าน `POST /api/v1/auth/login` (public) ด้วย email + password
- **So that** ได้รับ JWT สำหรับเรียก guarded endpoints
- **Priority**: High
- **Source**: D1-4, requirement.md §2

**Acceptance Criteria (EARS)**:
1. WHEN ระบบตรวจสอบ credentials (email + password) แล้วถูกต้อง THEN ระบบ SHALL ตอบ JWT ที่ sign ด้วย HMAC HS256
2. WHEN email หรือ password ไม่ถูกต้อง THEN ระบบ SHALL ปฏิเสธด้วย error invalid credentials (โดยไม่ระบุว่าส่วนไหนผิด)
3. WHEN JWT ถูก sign THEN ระบบ SHALL ใช้ secret key ที่เข้าถึงผ่าน config/env

#### US-004: JWT Middleware ปกป้อง endpoints
- **As a** API consumer
- **I want** guarded operations (list, get, update, delete) ให้ต้องการ Bearer JWT
- **So that** เฉพาะ client ที่มี token ถูกต้องเท่านั้นเข้าถึงได้
- **Priority**: High
- **Source**: D1-4, requirement.md §2

**Acceptance Criteria (EARS)**:
1. WHEN request เข้ามาที่ guarded endpoint พร้อม `Authorization: Bearer <valid JWT>` THEN ระบบ SHALL ปล่อยให้เข้าถึง endpoint นั้น
2. WHEN request เข้ามาโดยไม่มีหรือมี JWT ไม่ถูกต้อง/หมดอายุ THEN ระบบ SHALL ตอบ 401 Unauthorized
3. WHERE guarded endpoint ถูกเรียก THEN endpoint นั้น SHALL require token ยกเว้น `POST /api/v1/users` และ `POST /api/v1/auth/login` และ `GET /health`

### Area B: User Operations

#### US-005: ดึงผู้ใช้โดย ID / รายการผู้ใช้
- **As a** API consumer
- **I want** เรียก `GET /api/v1/users/:id` และ `GET /api/v1/users`
- **So that** ดึง user รายบุคคลหรือรายการผู้ใช้ทั้งหมดได้ (ต้องมี JWT)
- **Priority**: High
- **Source**: D1-4, requirement.md §3

**Acceptance Criteria (EARS)**:
1. WHEN เรียก `GET /api/v1/users/:id` ด้วย ID ที่มีอยู่ (มี JWT) THEN ระบบ SHALL ตอบข้อมูลผู้ใช้นั้น
2. WHEN เรียก `GET /api/v1/users/:id` ด้วย ID ที่ไม่มีอยู่ THEN ระบบ SHALL ตอบ 404 Not Found
3. WHEN เรียก `GET /api/v1/users` (มี JWT) THEN ระบบ SHALL ตอบรายการผู้ใช้ทั้งหมด

#### US-006: อัปเดตและลบผู้ใช้
- **As a** API consumer
- **I want** อัปเดต name/email (`PUT /api/v1/users/:id`) และลบผู้ใช้ (`DELETE /api/v1/users/:id`)
- **So that** ข้อมูลผู้ใช้ถูกจัดการได้ครบ lifecycle
- **Priority**: High
- **Source**: D1-4, requirement.md §3

**Acceptance Criteria (EARS)**:
1. WHEN อัปเดต name/email ของ user ที่มีอยู่ (มี JWT) THEN ระบบ SHALL อัปเดตข้อมูลและตอบผลลัพธ์
2. WHEN อัปเดตไปที่ email ที่ซ้ำกับ user อื่น THEN ระบบ SHALL ปฏิเสธด้วย error duplicate email
3. WHEN ลบ user ที่มีอยู่ (มี JWT) THEN ระบบ SHALL ลบออกจากระบบและตอบ success
4. WHEN อัปเดต/ลบด้วย ID ที่ไม่มีอยู่ THEN ระบบ SHALL ตอบ 404 Not Found

### Area C: Persistence

#### US-007: MongoDB Integration
- **As a** API consumer
- **I want** ข้อมูลผู้ใช้ถูกเก็บและอ่านจาก MongoDB ผ่าน official Go driver
- **So that** persistence ทำได้จริงและมี unique constraint บน email
- **Priority**: High
- **Source**: D1-6/8, requirement.md §4

**Acceptance Criteria (EARS)**:
1. WHEN user ถูกบันทึก THEN ระบบ SHALL ใช้ official Go MongoDB driver ในการ persist ไปยัง MongoDB
2. WHEN users collection ถูก initialize THEN ระบบ SHALL สร้าง unique index บน `email`
3. WHEN app ต้องการอ่าน/ค้น/เขียน user THEN ระบบ SHALL กระทำผ่าน Mongo repository abstraction (ไม่ฝัง driver logic ใน layer อื่น)

### Area D: Observability & Operations

#### US-008: Logging Middleware
- **As a** API consumer / operator
- **I want** ทุก HTTP request ถูกล็อกด้วย method, path และ execution time
- **So that** ตรวจสอบการใช้งานและแก้ปัญหาได้
- **Priority**: High
- **Source**: D1-4, requirement.md §5

**Acceptance Criteria (EARS)**:
1. WHEN request ผ่านเข้ามาใน HTTP server THEN ระบบ SHALL log method, path, status และ execution time ของ request นั้น
2. WHEN response เสร็จสิ้น THEN ระบบ SHALL log ความสำเร็จของ middleware โดยไม่รบกวน response payload

#### US-009: Concurrency Task (นับผู้ใช้ทุก 10 วินาที)
- **As a** operator
- **I want** background goroutine ที่รันทุก ๆ 10 วินาทีเพื่อนับและ log จำนวน users ทั้งหมดใน database
- **So that** เฝ้าดูขนาดข้อมูลได้แบบเรียลไทม์
- **Priority**: High (core), Medium effort
- **Source**: D1-4/8, requirement.md §6, requirement_brd.md (deep-dive)

**Acceptance Criteria (EARS)**:
1. WHEN server เริ่มทำงาน THEN ระบบ SHALL รัน foreground start logging และทำซ้ำทุก ๆ 10 วินาที
2. WHEN ครบ 10 วินาที THEN ระบบ SHALL log จำนวน total users ผ่าน collection-level count
3. WHEN ระบบได้รับสัญญาณ shutdown THEN ระบบ SHALL หยุด task อย่างถูกต้อง (ticker หยุด, goroutine จบ)
4. WHEN counting มี error THEN ระบบ SHALL log error และนับรอบถัดไปต่อไป (ไม่แครช)

### Area E: Quality

#### US-010: หน่วยทดสอบ (Unit Testing)
- **As a** developer
- **I want** unit tests ที่ใช้ Go standard `testing` package + mock MongoDB
- **So that** โค้ดถูกตรวจสอบความถูกต้องและเป็นไปตาม evaluation criteria (test coverage)
- **Priority**: High
- **Source**: D1-8, requirement.md §7

**Acceptance Criteria (EARS)**:
1. WHEN รัน `make test` THEN ระบบ SHALL รันทุก unit test ผ่านเกณฑ์ coverage ที่ตั้งไว้
2. WHERE โค้ดต้องทดสอบกับ MongoDB THEN ระบบ SHALL ใช้ mock/fake repository แทนการต่อ database จริง
3. WHEN มี test suite THEN suite นั้น SHALL ครอบคลุม service layer, handlers, jwt/bcrypt adapters และ helpers

### Area F: Bonus

#### US-011: Containerization (Docker + docker-compose)
- **As a** consumer/operator
- **I want** รัน API + MongoDB ผ่าน Docker และ docker-compose
- **So that** setup/execution ง่ายและ reproducibility ดี
- **Priority**: Low
- **Source**: D1-6/9, requirement.md Bonus

**Acceptance Criteria (EARS)**:
1. WHEN รัน `docker-compose up` THEN ระบบ SHALL เริ่ม MongoDB และ API พร้อมกัน (มี healthcheck)
2. WHERE image ถูก build THEN ระบบ SHALL ใช้ multi-stage build (build image แยกจาก runtime)

#### US-012: Interface Abstraction
- **As a** developer
- **I want** MongoDB operations abstract ผ่าน Go interfaces
- **So that** โค้ดเทสต์ง่ายและสลับ driver/storage ได้
- **Priority**: Medium
- **Source**: D1-6, requirement.md Bonus, tech-stack.md

**Acceptance Criteria (EARS)**:
1. WHEN domain/application ต้องการ persistence THEN domain/application SHALL อ้างอิง interface port เท่านั้น
2. WHERE port ถูก implement กับ MongoDB THEN การ implement SHALL อยู่ที่ adapter layer

#### US-013: Input Validation
- **As a** API consumer
- **I want** ระบบ validate input (required fields, email format)
- **So that** ข้อมูลเสียไม่เข้าสู่ระบบ
- **Priority**: Medium
- **Source**: D1-7/8, requirement.md Bonus

**Acceptance Criteria (EARS)**:
1. WHEN input ว่าง/ขาด required field THEN ระบบ SHALL ตอบ validation error พร้อมระบุ field
2. WHEN email มี format ไม่ถูกต้อง THEN ระบบ SHALL ปฏิเสธก่อน persist
3. WHEN body มีขนาดเกิน limit THEN ระบบ SHALL ปฏิเสธ request

#### US-014: Graceful Shutdown
- **As a** operator
- **I want** ให้ระบบจบการทำงานอย่างเป็นระเบียบเมื่อได้รับสัญญาณ (SIGINT/SIGTERM) ผ่าน `context.Context`
- **So that** ไม่ทิ้งงานค้างและปิด connection ถูกต้อง
- **Priority**: Medium
- **Source**: D1-8, requirement.md Bonus

**Acceptance Criteria (EARS)**:
1. WHEN ได้รับ `SIGINT` หรือ `SIGTERM` THEN ระบบ SHALL หยุดรับงานใหม่, ปิด HTTP server และ gRPC server แบบ graceful
2. WHEN shutdown เริ่ม THEN ระบบ SHALL ปิด MongoDB connection และหยุด background task

#### US-015: gRPC Support (CreateUser / GetUser)
- **As a** API consumer
- **I want** เข้าถึง `CreateUser` และ `GetUser` ผ่าน gRPC
- **So that** มีช่องทางเข้าถึงแบบ high-performance พร้อม auth ผ่าน token metadata
- **Priority**: Medium
- **Source**: D1-6, requirement.md Bonus, tech-stack.md

**Acceptance Criteria (EARS)**:
1. WHEN gRPC client เรียก `CreateUser` THEN ระบบ SHALL สร้าง user (เช่นเดียวกับ REST register, public)
2. WHEN gRPC client เรียก `GetUser` โดยไม่มีหรือ token ไม่ถูกต้องใน metadata THEN ระบบ SHALL ปฏิเสธ (Unauthenticated)
3. WHEN gRPC client เรียก `GetUser` ด้วย token ถูกต้อง THEN ระบบ SHALL ตอบ user ตาม ID
4. WHEN domain error เกิดใน gRPC flow THEN ระบบ SHALL map error เป็น gRPC code ที่ตรงกัน (NotFound/AlreadyExists/InvalidArgument/Unauthenticated/Internal)

#### US-016: Hexagonal Architecture
- **As a** developer
- **I want** โครงสร้างโปรเจกต์แบบ ports & adapters แยก domain/application/infrastructure
- **So that** business logic ถูก decouple จาก frameworks/drivers
- **Priority**: Low
- **Source**: D1-1/6/8, requirement.md Bonus

**Acceptance Criteria (EARS)**:
1. WHERE package อยู่ใน domain layer THEN package นั้น SHALL ไม่ import Fiber/HTTP/MongoDB
2. WHEN application layer ทำงาน THEN มัน SHALL เรียกผ่าน ports (interfaces) เท่านั้น ไม่เรียก framework/library โดยตรง
3. WHEN เปลี่ยน framework (เช่น Fiber → Gin) THEN domain และ application SHALL เปลี่ยนแปลงน้อยที่สุด

## Story Summary

| ID | Title | Area | Priority | Dependencies |
|----|-------|------|----------|--------------|
| US-001 | User Model | Auth & Model | High | — |
| US-002 | Registration | Auth & Model | High | US-001 |
| US-003 | Login → JWT | Auth & Model | High | US-001 |
| US-004 | JWT middleware | Auth & Model | High | US-003 |
| US-005 | Get by ID / List | User Operations | High | US-001, US-004 |
| US-006 | Update / Delete | User Operations | High | US-001, US-004 |
| US-007 | MongoDB integration | Persistence | High | US-001 |
| US-008 | Logging middleware | Observability | High | — |
| US-009 | Concurrency task (10s) | Observability | High | US-007 |
| US-010 | Unit testing | Quality | High | — |
| US-011 | Containerization | Bonus | Low | US-007 |
| US-012 | Interface abstraction | Bonus | Medium | US-007 |
| US-013 | Input validation | Bonus | Medium | US-002 |
| US-014 | Graceful shutdown | Bonus | Medium | — |
| US-015 | gRPC support | Bonus | Medium | US-001, US-004 |
| US-016 | Hexagonal architecture | Bonus | Low | — |

## Non-Functional Notes
- **Security**: JWT sign ด้วย HMAC HS256 (secret จาก config/env); password hashed ด้วย bcrypt; guarded endpoints ต้อง Authen ทุกครั้ง; ระวังไม่ leak ว่า credentials ส่วนไหนผิด
- **Performance**: concurrency count task ใช้ `EstimatedDocumentCount` (เร็ว, near-perfect accuracy สำหรับ collection ที่มีการลบน้อย); MongoDB ใช้ unique index บน email
- **Testability**: ports & adapters ช่วยให้ mock ได้ (gomock) ทั้ง user repo/statusdb/hasher/token
- **Maintainability**: hexagonal layering — domain ไม่รู้จัก framework; thinning adapter layer

## External References

| Source | Stories | What was used |
|--------|---------|---------------|
| `documents/requirements/user-management-api/requirement.md` | US-001..US-010 | ข้อกำหนด core ทั้ง 7 ข้อ + bonus 6 ข้อ + evaluation criteria |
| `documents/requirements/user-management-api/requirement_brd.md` | US-009, US-015 | Deep-dive concurrency task, gRPC design/error mapping, สถานะตาม requirement |
| `documents/requirements/user-management-api/tech-stack.md` | US-012, US-015, US-016 | โครงสร้าง hexagonal, ports/adapter, endpoint mapping, DI, tooling |
| `documents/requirements/README.md` | US-001..US-010 | ภาพรวม Path 1 (User Management API) งานหลัก + bonus |