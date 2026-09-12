# Requirements Decisions (D1)

## Context Summary

- **Type**: Brownfield — โปรเจกต์ Go ที่มี implement ทั้ง feature แล้ว (Response: RESTful API จัดการผู้ใช้)
- **Scope**: new · **Complexity**: Medium · 13 stories (7 core + 6 bonus), 1 domain (user management), 1 user type
- **Stack**: Go 1.26.4 / Fiber v2 / MongoDB (mongo-driver v2) / JWT (HMAC HS256) / gRPC / bcrypt
- **Architecture**: Hexagonal (Ports & Adapters) — `adapter → application → domain`
- **Feature**: User Management API — register/login (JWT), CRUD users, logging middleware, background count task ทุก 10 วิ, testing
- **Requirements source**: `documents/requirements/user-management-api/requirement.md` (7 ข้อบังคับ + 6 bonus) + `requirement_brd.md` + `tech-stack.md`
- **Recommendations**: Personas No · Units No · NFR Yes

---

## Decision Questions

### D1-1: ขอบเขตของ feature
**Question**: ขอบเขตการใช้งานของ "User Management API" นี้ควรเป็นแบบใด?
- 1) สร้าง full product ตาม requirement.md ครบทั้ง 7 ข้อบังคับ (model, auth, CRUD, MongoDB, middleware, concurrency task, testing) **(Recommended)**
- 2) สร้างเฉพาะ MVP (model + auth + CRUD) ไว้ขยายทีหลัง
- 3) สร้าง full product + bonus ทั้งหมด (Docker, validation, graceful shutdown, gRPC, hexagonal)
- 4) Other (please specify): _______

**Answer**: 1) สร้าง full product ตาม requirement.md ครบทั้ง 7 ข้อบังคับ (model, auth, CRUD, MongoDB, middleware, concurrency task, testing)

---

### D1-2: ผู้ใช้ระบบ (User Types)
**Question**: ใครเป็นผู้ใช้ระบบนี้บ้าง?
- 1) API consumer/backend integrator แบบเดียว (ทีมหรือผู้เรียกใช้ REST/gRPC endpoints) **(Recommended)**
- 2) ผู้ใช้ปลายทางสองกลุ่ม (ลูกค้า + ผู้ดูแลระบบ)
- 3) มากกว่าสองกลุ่ม (รวม service-to-service automation)
- 4) Other (please specify): _______

**Answer**: 1) API consumer/backend integrator แบบเดียว (ทีมหรือผู้เรียกใช้ REST/gRPC endpoints)

---

### D1-3: Personas
**Question**: ต้องการสร้าง personas.md สำหรับอธิบายผู้ใช้แต่ละแบบหรือไม่?
- 1) ไม่ — ผู้ใช้ระบบมีแบบเดียว (API consumer) ไม่คุ้มกับ document แยก **(Recommended)**
- 2) ใช่ — สร้าง personas.md สำหรับผู้ใช้ทุกกลุ่ม
- 3) Other (please specify): _______

**Answer**: 1) ไม่ — ผู้ใช้ระบบมีแบบเดียว (API consumer) ไม่คุ้มกับ document แยก

---

### D1-4: ฟังก์ชันหลัก (Core Functionality)
**Question**: ฟังก์ชันหลักที่ต้องมีในระบบคืออะไรบ้าง? (เลือกได้หลายข้อ)
- 1) User Model: ID, Name, Email (unique), Password (hashed), CreatedAt **(Recommended)**
- 2) Authentication: register + login คืน JWT (HMAC HS256) + middleware ปกป้อง endpoints **(Recommended)**
- 3) User Operations: create, get by ID, list, update name/email, delete **(Recommended)**
- 4) MongoDB integration + logging middleware + concurrency task (นับ users ทุก 10 วิ) + testing **(Recommended)**
- 5) Other (please specify): _______

**Answer**: 1) User Model: ID, Name, Email (unique), Password (hashed), CreatedAt; 2) Authentication (register + login JWT HMAC HS256 + middleware); 3) User Operations (create/get/list/update/delete); 4) MongoDB integration + logging middleware + concurrency task + testing

---

### D1-5: Data Entities
**Question**: ข้อมูลหลักของระบบคือ User entity — ควรยืนยันฟิลด์ใดเป็นส่วนประกอบหลัก?
- 1) `ID` (auto-generated), `Name`, `Email` (unique), `Password` (hashed), `CreatedAt` — ตาม spec **(Recommended)**
- 2) เพิ่มฟิลด์เพิ่มเติม เช่น `UpdatedAt`, `Role`, `Status`
- 3) ใช้โครงสร้างอื่นตามดุลยพินิจ (ระบุ)
- 4) Other (please specify): _______

**Answer**: 1) `ID` (auto-generated), `Name`, `Email` (unique), `Password` (hashed), `CreatedAt` — ตาม spec

---

### D1-6: Integrations
**Question**: มี integration ภายนอกอะไรบ้าง และตัวใดจำเป็นต้องเป็น core (ไม่มีได้ใน MVP)?
- 1) MongoDB (persistence) — core; JWT/Bcrypt (internal lib) — core; gRPC — เป็น bonus ไม่จำเป็นใน MVP **(Recommended)**
- 2) MongoDB + JWT + gRPC ทุกตัวเป็น core
- 3) ใช้เฉพาะ MongoDB + JWT; ตัด gRPC ออกทั้งหมด
- 4) Other (please specify): _______

**Answer**: 1) MongoDB (persistence) — core; JWT/Bcrypt (internal lib) — core; gRPC — เป็น bonus ไม่จำเป็นใน MVP

---

### D1-7: Business Rules
**Question**: กฎทางธุรกิจใดของ system นี้ควรบังคับใช้?
- 1) email ต้อง unique + normalize เป็น lowercase + ตรวจ format; password ต้องผ่าน hashing (bcrypt); password min 8 chars **(Recommended)**
- 2) กฎเพิ่มเติม: อัปเดต email ต้องตรวจ unique ซ้ำ; ห้ามลบ user ที่อ้างอิงจากภายนอก
- 3) ใช้กฎตาม requirement.md อย่างเคร่งครัดเท่านั้น
- 4) Other (please specify): _______

**Answer**: 1) email ต้อง unique + normalize เป็น lowercase + ตรวจ format; password ต้องผ่าน hashing (bcrypt); password min 8 chars

---

### D1-8: ข้อจำกัด (Constraints)
**Question**: ข้อจำกัดด้านเทคนิค/การส่งมอบที่ต้องยึดถือ?
- 1) Go standard `testing` package + mock MongoDB; ใช้ official Go MongoDB driver; JWT HMAC HS256; concurrency task รันทุก 10 วิ; code ให้สะอาด/อ่านง่าย (evaluation criteria) **(Recommended)**
- 2) จำกัดเฉพาะ standard library (ห้าม library ภายนอก)
- 3) มีข้อจำกัดเพิ่มเติม (ระบุ)
- 4) Other (please specify): _______

**Answer**: 1) Go standard `testing` package + mock MongoDB; ใช้ official Go MongoDB driver; JWT HMAC HS256; concurrency task รันทุก 10 วิ; code ให้สะอาด/อ่านง่าย (evaluation criteria)

---

### D1-9: ลำดับความสำคัญ (Priorities)
**Question**: อะไรคือ must-have เทียบกับ nice-to-have สำหรับ deliverable ชุดนี้?
- 1) 7 core requirements = must-have; bonus (Docker, validation, graceful shutdown, gRPC, hexagonal) = nice-to-have **(Recommended)**
- 2) ทุกอย่างรวม bonus เป็น must-have
- 3) เฉพาะ core ข้อ 1–5 เท่านั้นเป็น must-have
- 4) Other (please specify): _______

**Answer**: 1) 7 core requirements = must-have; bonus (Docker, validation, graceful shutdown, gRPC, hexagonal) = nice-to-have

---

### D1-10: ขนาดทีม
**Question**: How many developers will work on this project?
- 1) Solo (1 developer)
- 2) Small team (2–3)
- 3) Medium team (4–8)
- 4) Large team (9+)
- 5) Other (please specify): _______

**Answer**: 1) Solo (1 developer)

---

## Decisions Summary
<!-- Machine-readable compact summary. Downstream phases: read ONLY this section. -->
<!-- Auto-populated after user fills answers. One line per decision. -->
- D1-1 Feature scope: Full product ตาม requirement.md ครบ 7 ข้อบังคับ (1)
- D1-2 User types: API consumer/backend integrator แบบเดียว (1)
- D1-3 Personas: ไม่สร้าง personas.md (1)
- D1-4 Core functionality: Model + Auth (JWT) + CRUD + MongoDB/middleware/concurrency/testing (1–4)
- D1-5 Data entities: ID, Name, Email (unique), Password (hashed), CreatedAt (1)
- D1-6 Integrations: MongoDB core; JWT/Bcrypt core; gRPC bonus (1)
- D1-7 Business rules: email unique + lowercase + format; bcrypt; password min 8 (1)
- D1-8 Constraints: Go testing + mock MongoDB; official driver; HS256; 10 วิ; clean code (1)
- D1-9 Priorities: 7 core = must-have; bonus = nice-to-have (1)
- D1-10 Team size: Solo (1 developer) (1)

---

**Instructions**: Fill in your answers above and respond with "done"