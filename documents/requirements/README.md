# Requirements Summary

โจทย์จาก README.md หลักของโปรเจกต์แบ่งเป็น 2 ส่วนงาน (2 Paths):

| Path | หัวข้อ | ประเภท Submission | ไฟล์ Requirements |
| --- | --- | --- | --- |
| Path 1 | User Management API | Code implementation (Golang) | `user-management-api/requirement.md` |
| Path 2 | Lottery Search System | Design proposal only (ไม่ต้องเขียนโค้ด) | `lottery-search-system/requirement.md` |

---

## Path 1: User Management API (เขียนโค้ด)

สร้าง RESTful API ด้วย **Golang** ใช้ **MongoDB** สำหรับ persistence และ **JWT** สำหรับ authentication

### งานหลัก
- **User Model**: `ID`, `Name`, `Email` (unique), `Password` (hashed), `CreatedAt`
- **Authentication**: register + login (คืน JWT), sign ด้วย HMAC `HS256`
- **User Operations**: CRUD — create, get by ID, list, update name/email, delete
- **MongoDB Integration**: ใช้ official Go MongoDB driver
- **Middleware**: logging middleware (method, path, execution time)
- **Concurrency Task**: background goroutine ทุก 10 วินาที log จำนวน users ทั้งหมด
- **Testing**: unit tests ด้วย Go standard `testing` package + mock MongoDB

### Bonus (Optional)
- Docker / docker-compose
- Interface abstraction สำหรับ MongoDB
- Input validation
- Graceful shutdown ด้วย `context.Context`
- gRPC (`CreateUser`, `GetUser`)
- Hexagonal architecture (ports & adapters)

### Deliverables
- `README.md` พร้อม setup/execution instructions
- คู่มือการ generate/use JWT
- ตัวอย่าง API requests/responses
- Documentation ของ assumptions/design decisions

---

## Path 2: Lottery Search System (Design Proposal เท่านั้น)

ออกแบบระบบค้นหา lottery ticket (6 หลัก) ขนาด **10 ล้านรายการ** ด้วย wildcard pattern matching

### งานหลัก (design only, ไม่มีโค้ด)
- **Data Volume**: 10M tickets, แต่ละตัวเป็นเลข 6 หลัก
- **Search Pattern**: 6 ตัวอักษร ประกอบด้วย digit + `*` เช่น `****23`, `1****5`, `123***`
- **Result Distribution**: ป้องกันไม่ให้ ticket เดียวถูก return ให้หลาย user พร้อมกัน
- **Performance**: ค้นหาได้เร็วที่ 10M+ records
- **Real-World Design**: แนะนำ database/storage, algorithm/indexing strategy, กลไกป้องกัน duplicate (locking / reservation / atomic allocation)

### Deliverables (เอกสารอย่างเดียว)
- Solution architecture, data structures, algorithms
- Production database/storage ที่แนะนำพร้อมเหตุผล
- Performance analysis (efficiency + tradeoffs)
- Concurrency/distribution strategy

---

## สถานะปัจจุบัน
- [x] `documents/requirements/user-management-api/requirement.md` — สร้างแล้ว
- [x] `documents/requirements/lottery-search-system/requirement.md` — สร้างแล้ว
- [ ] Implement Path 1 (User Management API)
- [ ] เขียน Design Proposal Path 2 (Lottery Search System)