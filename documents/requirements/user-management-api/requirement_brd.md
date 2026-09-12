# User Management API — Requirement BRD

## ภาพรวม

RESTful API สร้างด้วย **Golang** สำหรับจัดการผู้ใช้ ใช้ **MongoDB** เป็น persistence และ **JWT** สำหรับ authentication ตาม requirements ใน `requirement.md`

เน้นบันทึกนี้: อธิบายการ implement **Requirement ข้อ 6 — Concurrency Task** พร้อมคำสั่ง MongoDB ที่ใช้

---

## Requirement 6: Concurrency Task

> Run a background goroutine every 10 seconds to log the total number of users in the database

### สิ่งที่ต้องทำ
- รัน background goroutine ทุก ๆ **10 วินาที**
- log จำนวนผู้ใช้ทั้งหมด (total users) ใน database
- ต้องสามารถหยุดได้อย่างถูกต้อง (graceful shutdown)

### สถาปัตยกรรม (ปัจจุบัน)

ใช้ **Hexagonal Architecture** แยกเป็น 3 ชั้น ไม่ต่างจาก user CRUD:

| ชั้น | ไฟล์ (ปัจจุบัน) | บทบาท |
| --- | --- | --- |
| Domain port | `internal/domain/statusdb/repository.go` | interface `Repository { Count(ctx) (int64, error) }` |
| Application | `internal/application/services/statusdb/service.go` | use case `Service.Count` (wrapper ไปยัง port) |
| Outbound adapter | `internal/adapter/outbound/mongodb/user_repository.go` | implement `Count` ด้วย `EstimatedDocumentCount` |
| Composition | `cmd/api/main.go` → `countUsersLoop` | background goroutine + graceful shutdown |

จุดเด่น: use case ไม่รู้จัก MongoDB — มันเห็นแค่ `Repository` port
(ทดสอบด้วย gomock ที่ `internal/mocks/statusdb/`)

### คำสั่ง MongoDB ที่ใช้ — เปรียบเทียบ 3 วิธี

| วิธี | คำสั่ง | ความแม่นยำ | ความเร็ว | วิธีทำงาน |
| --- | --- | --- | --- | --- |
| **CountDocuments** | `collection.CountDocuments(ctx, bson.M{})` | แม่นยำ 100% | ช้า (collection scan) | สแกนเอกสารทุกตัวใน collection แล้วนับ |
| **EstimatedDocumentCount** | `collection.EstimatedDocumentCount(ctx)` | โดยประมาณ (.metadata-based) | เร็วมาก | ใช้ค่า metadata จาก storage engine ไม่ scan |
| **CountDocuments + hint** | `collection.CountDocuments(ctx, bson.M{}, options.Count().SetHint("_id_"))` | แม่นยำ 100% | เร็ว (index scan) | นับผ่าน index `_id_` แทน collection scan |

**Source:** [MongoDB Go Driver — Count Documents](https://www.mongodb.com/docs/drivers/go/v1.x/fundamentals/crud/read-operations/count)

### การตัดสินใจ (Decision)

โจทย์นี้เลือก **`EstimatedDocumentCount()`** เพราะ:

- เป็น job poll สำหรับ **logging** ไม่ใช่ billing → ความเร็วสำคัญกว่าความแม่นยำหลักหน่วย
- แม่นยำ 100% สำหรับ collection ที่ไม่มี document ถูกลบ (กรณี use case นี้ users มักถูกเพิ่มอย่างเดียว)
- เหมาะกับ monitoring/logging ทุกๆ 10 วินาที

### Implementation จริง

Port (`internal/domain/statusdb/repository.go`):

```go
package statusdb

import "context"

// Repository is the port the application needs for collection-level statistics.
type Repository interface {
	Count(ctx context.Context) (int64, error)
}
```

Use case (`internal/application/services/statusdb/service.go`):

```go
type Service struct {
	repo domainstatusdb.Repository
}

func NewService(repo domainstatusdb.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
```

Outbound adapter (`internal/adapter/outbound/mongodb/user_repository.go`):

```go
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	n, err := r.collection.EstimatedDocumentCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}
```

Background goroutine (`cmd/api/main.go`):

```go
go countUsersLoop(ctx, statusdb.NewService(userRepo), logger)

func countUsersLoop(ctx context.Context, svc *statusdb.Service, logger *slog.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	logger.Info("user count task started")
	for {
		select {
		case <-ctx.Done():
			logger.Info("user count task stopped")
			return
		case <-ticker.C:
			total, err := svc.Count(ctx)
			if err != nil {
				logger.Error("failed to count users", "error", err)
				continue
			}
			logger.Info("total users", "count", total)
		}
	}
}
```

ตัวอย่างผลลัพธ์ log ที่ได้ (ทุกๆ 10 วินาที):

```
time=2026/09/13 01:00:00 level=INFO msg="total users" count=104203
time=2026/09/13 01:00:10 level=INFO msg="total users" count=104215
time=2026/09/13 01:00:20 level=INFO msg="total users" count=104220
```

### Testing

| ชั้น | วิธี | ไฟล์ |
| --- | --- | --- |
| `statusdb.Service` | gomock mock ของ `Repository` (test ทั้ง happy path + error) | `internal/application/services/statusdb/service_test.go` |
| mongo `Count` | integration test ผ่าน `internal/infrastructure/db` (`make test-integration`) | `mongodb/user_repository_integration_test.go` |

### หมายเหตุการออกแบบ
- แยก port ใหม่ (`statusdb.Repository`) ออกจาก user repository เพื่อให้ user CRUD service โฟกัสเฉพาะ user flows
- ใช้ `time.NewTicker` แทน `time.Sleep` เพื่อความแม่นยำของ interval 10 วินาที และปิด `ticker` เมื่อจบ
- รันผ่าน `go countUsersLoop(ctx, ...)` และรับสัญญาณหยุดจาก `signal.NotifyContext(os.Interrupt, syscall.SIGTERM)` (SIGINT/SIGTERM) → ลูปจบเมื่อ `ctx.Done()`
- เหลือรายละเอียดที่เลือก **ไม่** ใช้ในโปรเจกต์นี้: `CountDocuments` + hint `_id_` / filter `MinKey` (ดู Performance Tip ด้านล่าง) — เก็บเป็นตัวเลือกไว้ถ้าต้องการความแม่นยำ 100%

### Performance Tip (สำรอง ถ้าต้องการความแม่นยำ)

ใช้ filter `{ "_id": { $gte: MinKey } }` เพื่อบังคับให้ MongoDB ใช้ `_id_` index:

```go
filter := bson.M{"_id": bson.M{"$gte": primitive.MinKey}}
total, err := collection.CountDocuments(ctx, filter)
```

- `primitive.MinKey` ต้อง import จาก `go.mongodb.org/mongo-driver/bson/primitive`
- วิธีนี้ได้ความแม่นยำเท่ากับ `CountDocuments` ปกติ แต่เร็วกว่าเพราะไม่ต้อง scan ทั้ง collection
- ใช้ได้เฉพาะกับ `CountDocuments` ไม่ใช่ `EstimatedDocumentCount`

---

## สรุปงานทั้งหมด (Path 1)

1. User Model (ID, Name, Email unique, Password hashed, CreatedAt)
2. Authentication — register/login, JWT HMAC `HS256`, middleware
3. User Operations — CRUD
4. MongoDB Integration — official Go driver
5. Logging Middleware — method, path, execution time
6. Concurrency Task — background goroutine ทุก 10 วินาที (count users ด้วย `EstimatedDocumentCount` ผ่าน `statusdb` port + graceful shutdown)
7. Testing — standard `testing` package + gomock + MongoDB integration test

Bonus: Docker, interface abstraction, validation, graceful shutdown, gRPC, hexagonal architecture