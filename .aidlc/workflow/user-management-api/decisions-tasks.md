# Tasks Decisions

## Context Summary
- **Phase**: Tasks (D4) — follow-up หลัง design approved
- **Environment**: Brownfield — โค้ด Go (Fiber v2 / gRPC / MongoDB) implement ครบแล้วใน repo
- **Design**: 12 components, 1 entity (User), 7 REST endpoints + 2 gRPC, 1 external integration (MongoDB) + JWT in-process
- **Team**: Solo (1 developer) — scope new, comprehensive
- **D3 หลัก**: Table-driven test (no PBT), opt-in live + bufconn + mock mongo, structured logging only
- **งาน tasks จะ bridge ระหว่าง design ที่ approved กับโค้ด/การ verify จริง**

---

## Decision Questions

### D4-1: Task Breakdown Strategy
**Question**: จะแบ่งงานจาก design ยังไง (12 components)?
- 1) 1 component = 1 task (ชัดเจน, match design/components.md)
- 2) แบ่งตาม feature/module (เช่น auth, user CRUD, infra) — รวมหลาย components ต่อ task **(Recommended)**
- 3) แบ่งตามเลเยอร์ (domain/application/adapters) — งานใหญ่ต่อเรื่อง
- 4) Other (please specify): _______

**Answer**: 2) Feature/module breakdown — แต่ละ task ครอบกลุ่ม components ของ feature (auth, CRUD, infra) ตาม design

---

### D4-2: Implementation Approach
**Question**: โค้ดมีอยู่แล้ว — เทคนิคการ implement เดิมคืออะไร?
- 1) Test-after — เขียนการทดสอบแล้ว refactor/ปรับโค้ดตาม **(Recommended)**
- 2) TDD เต็มรูปแบบ — เขียน test ก่อนทุกเรื่อง
- 3) Review-only — โค้ดเสร็จแล้ว แค่ตรวจสอบกับ design
- 4) Other (please specify): _______

**Answer**: 1) Test-after — เขียน/ปรับการทดสอบแล้ว refactor หรือปรับโค้ดตาม design

---

### D4-3: Component Priority
**Question**: ลำดับการทำงาน/verify ไหนก่อน?
- 1) Core domain/application (user service, domain) ก่อน แล้วค่อย adapters
- 2) ตาม user-facing flow (register → login → CRUD) — verify ผ่าน API
- 3) Adapters/structural (config, db connect) ก่อน แล้ว logic **(Recommended)**
- 4) Other (please specify): _______

**Answer**: 3) Adapters/structural (config, db connect) ก่อน แล้ว logic

---

### D4-4: Integration Strategy (Dependencies)
**Question**: จัดการ external dependencies (MongoDB) ระหว่างทำงานยังไง?
- 1) Mock ทั้งหมดระหว่าง dev, live test แยก
- 2) Real local MongoDB (docker-compose) ระหว่าง dev **(Recommended)**
- 3) Testcontainers/managed DB
- 4) Other (please specify): _______

**Answer**: 2) Real local MongoDB (docker-compose) ระหว่าง dev

---

### D4-5: Testing Strategy
**Question**: จะรวม test types ไหนใน task list?
- 1) Unit + integration (live opt-in) เท่านั้น **(Recommended)**
- 2) Unit + integration + API contract (OpenAPI)
- 3) Unit เท่านั้น (เร็ว)
- 4) Other (please specify): _______

**Answer**: 1) Unit + integration (live opt-in) เท่านั้น

---

### D4-6: Task Granularity
**Question**: ขนาด scope ต่อ 1 task?
- 1) Atomic — หนึ่งความคิด/ไฟล์ ต่อ task (เล็กมาก)
- 2) Single-concern — 1 ความรับผิดชอบ (เช่น "verify + enhance user service") **(Recommended)**
- 3) Multi-concern — กลุ่มที่เกี่ยวข้อง (เช่น "auth flow ทั้งหมด")
- 4) Other (please specify): _______

**Answer**: 2) Single-concern — 1 ความรับผิดชอบต่อ task

---

### D4-7: Execution Mode
**Question**: ลำดับการ execute tasks?
- 1) Sequential — ทีละตัว ตาม dependency
- 2) Parallel — หลาย tasks พร้อมกัน (อาศัย file ownership แยก) **(Recommended)**
- 3) Hybrid — คู่แรก parallel แล้วแตะ sequential
- 4) Other (please specify): _______

**Answer**: 2) Parallel — ออกแบบ execution waves ให้ทำงานพร้อมกันได้ โดยไม่มี file ownership ทับกัน

---

## Decisions Summary
<!-- Machine-readable compact summary. Downstream phases: read ONLY this section. -->
<!-- Auto-populated after user fills answers. One line per decision. -->
- D4-1 Breakdown: Feature/module breakdown (tasks ครอบกลุ่ม components ต่อ feature)
- D4-2 Approach: Test-after (เขียน/ปรับ tests แล้ว refactor ตาม design)
- D4-3 Priority: Adapters/structural (config, db) ก่อน logic
- D4-4 Integration: Real local MongoDB (docker-compose) ระหว่าง dev
- D4-5 Testing: Unit + integration (live opt-in) only
- D4-6 Granularity: Single-concern (1 ความรับผิดชอบต่อ task)
- D4-7 Execution: Parallel (execution waves, ไม่มี file ownership overlap)

---

**Instructions**: Fill in your answers above and respond with "done"