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

### คำสั่ง MongoDB ที่ใช้ — เปรียบเทียบ 2 วิธี

| วิธี | คำสั่ง | ความแม่นยำ | ความเร็ว | วิธีทำงาน |
| --- | --- | --- | --- | --- |
| **CountDocuments** | `collection.CountDocuments(ctx, bson.M{})` | แม่นยำ 100% | ช้า (collection scan) | สแกนเอกสารทุกตัวใน collection แล้วนับ |
| **EstimatedDocumentCount** | `collection.EstimatedDocumentCount(ctx)` | โดยประมาณ (.metadata-based) | เร็วมาก | ใช้ค่า metadata จาก storage engine ไม่ scan |
| **CountDocuments + hint** | `collection.CountDocuments(ctx, bson.M{}, options.Count().SetHint("_id_"))` | แม่นยำ 100% | เร็ว (index scan) | นับผ่าน index `_id_` แทน collection scan |

**Source:** [MongoDB Go Driver — Count Documents](https://www.mongodb.com/docs/drivers/go/v1.x/fundamentals/crud/read-operations/count)

### Performance Tip

ใช้ filter `{ "_id": { $gte: MinKey } }` เพื่อบังคับให้ MongoDB ใช้ `_id_` index:

```go
filter := bson.M{"_id": bson.M{"$gte": primitive.MinKey}}
total, err := collection.CountDocuments(ctx, filter)
```

- `primitive.MinKey` ต้อง import จาก `go.mongodb.org/mongo-driver/bson/primitive`
- วิธีนี้ได้ความแม่นยำเท่ากับ `CountDocuments` ปกติ แต่เร็วกว่าเพราะไม่ต้อง scan ทั้ง collection
- ใช้ได้เฉพาะกับ `CountDocuments` ไม่ใช่ `EstimatedDocumentCount`

### คำแนะนำ: ควรใช้วิธีไหนสำหรับ Job ทุก 10 วินาที?

- **ถ้าต้องการความแม่นยำทุกครั้ง** → ใช้ `CountDocuments` + hint `_id_` หรือ filter `MinKey` (เร็ว แม่น)
- **ถ้ายอมรับค่าโดยประมาณได้ (±1-2% error)** → ใช้ `EstimatedDocumentCount()` เร็วที่สุด เหมาะกับ monitoring/logging
- **คำแนะนำสำหรับโจทย์นี้**: ใช้ `EstimatedDocumentCount()` เพราะเป็น job poll สำหรับ logging ไม่ใช่ billing — ความเร็วสำคัญกว่าความแม่นยำหลักหน่วย

### โครงสร้างการ implement (แนวทาง)

1. **Background goroutine** — วนลูปด้วย `time.Ticker` หรือ `time.Sleep` ทุก 10 วินาที
2. **เรียก count method** — นับจำนวน users แล้ว log ด้วย `log.Printf` / `slog` (เลือกวิธีตามข้างบน: `CountDocuments` / `EstimatedDocumentCount`)
3. **Graceful shutdown** — ใช้ `context.Context` (`ctx.Done()`) เพื่อหยุดลูปเมื่อได้รับ signal (`SIGINT`/`SIGTERM`)

ตัวอย่างโครงสร้าง (แบบแม่นยำ — `CountDocuments`):

```go
func (s *Service) StartUserCountTicker(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Println("stopping user count ticker")
            return
        case <-ticker.C:
            total, err := s.repo.CountUsers(ctx)
            if err != nil {
                log.Printf("failed to count users: %v", err)
                continue
            }
            log.Printf("total users: %d", total)
        }
    }
}
```

ตัวอย่างผลลัพธ์ log ที่ได้ (ทุกๆ 10 วินาที):

```
2026/09/12 10:00:00 total users: 104,203
2026/09/12 10:00:10 total users: 104,215
2026/09/12 10:00:20 total users: 104,220
```

หากเลือกใช้แบบเร็ว (`EstimatedDocumentCount`) ให้เปลี่ยน method ใน repository เป็น:

```go
total, err := collection.EstimatedDocumentCount(ctx)
```

### หมายเหตุการออกแบบ
- แยก logic ผ่าน interface (`UserRepository`) เพื่อให้ test/mock ได้ง่าย
- ใช้ `Ticker` แทน `Sleep` เพื่อความแม่นยำของ interval 10 วินาที
- รัน goroutine ผ่าน `go s.StartUserCountTicker(ctx)` ตอน service start
- รอให้จบ (ส่ง `ctx` cancel / wait via `sync.WaitGroup` หรือ `errgroup`) ตอน shutdown

---

## สรุปงานทั้งหมด (Path 1)

1. User Model (ID, Name, Email unique, Password hashed, CreatedAt)
2. Authentication — register/login, JWT HMAC `HS256`, middleware
3. User Operations — CRUD
4. MongoDB Integration — official Go driver
5. Logging Middleware — method, path, execution time
6. Concurrency Task — background goroutine ทุก 10 วินาที (count users ด้วย `CountDocuments` หรือ `EstimatedDocumentCount`)
7. Testing — standard `testing` package + mock MongoDB

Bonus: Docker, interface abstraction, validation, graceful shutdown, gRPC, hexagonal architecture