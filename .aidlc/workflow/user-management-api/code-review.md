# Code Review — User Management API

- **วันที่**: 2026-09-13
- **ขอบเขต**: Full review (brownfield) — เจอ source ตามการอ่าน entry points → business logic → data access → auth → error handling → tests → config
- **ไฟล์ที่ตรวจ**: 26 ไฟล์ (main, config, application services, HTTP/gRPC inbound, middleware, outbound mongodb/jwt/bcrypt, domain, infrastructure)
- **อ้างอิง design**: `specs/user-management-api/design/{components,data-model,api-spec,testing-strategy}.md` + repo AGENTS.md
- **สถานะ compliance**: สอดคล้องเป็นส่วนใหญ่; ข้องอแค่ document-drift 3 จุด และ 3 ประเด็น major
- **Coverage**: 87.6% (non-generated) ≥ เกณฑ์ 80% ✅ · 84 tests / 15 packages ผ่านทั้งหมด ✅

## สรุป Findings

| Severity | จำนวน |
|----------|-------|
| 🔴 Critical | 0 |
| 🟡 Major | 3 |
| 🟢 Minor | 5 |
| 💡 Suggestion | 4 |

## สถานะหลังแก้ไข (2026-09-13 · re-review)

| Finding | สถานะ | หลักฐานการแก้ |
|---------|-------|----------------|
| CODE-MJ-1 | ✅ แก้แล้ว | `grpc/auth.go` — default → `codes.Internal "internal error"` + `slog.Default().Error` ฝั่ง server; test `TestCreateUser_DoesNotLeakInternalError` |
| CODE-MJ-2 | ✅ แก้แล้ว | HTTP: `middleware.RequireOwner` (403) ใน GET/PUT/DELETE `:id`; gRPC: subject==id ใน `GetUser` (`codes.PermissionDenied`); tests HTTP 403 + gRPC `TestGetUser_ForbiddenOnOtherID` |
| CODE-MJ-3 | ✅ แก้แล้ว | `config.load()` fail-fast เมื่อ `MONGO_URI`/`JWT_SECRET` ว่าง; test `TestLoadMissingRequiredSecrets` |
| CODE-MN-1 | ✅ แก้แล้ว | `helpers.WriteServiceError` คืน `err.Error()` เป็น top-level + อัปเดต api-spec.md; tests อัปเดต |
| CODE-MN-2 | ✅ แก้แล้ว | อัปเดต api-spec.md proto ให้ return `User` ตรงๆ |
| CODE-MN-3 | ✅ แก้แล้ว | data-model.md default → `userdb` |
| CODE-MN-4 | ✅ แก้แล้ว | `main.go` ใช้ `redactURI` (test `TestRedactURI`); credential ไม่ลง log |
| CODE-MN-5 | ✅ แก้แล้ว | ใช้ `userv1.UserService_GetUser_FullMethodName` จาก protoc (ตัด string literal) |
| CODE-SG-1 | ✅ แก้แล้ว | `middleware.RateLimit` + wire ที่ `/auth/login` (10 ครั้ง/นาที/IP → 429); tests middleware + `TestLogin_RateLimited` |
| CODE-SG-2 | ⏸️ เลื่อน | การ revoke/refresh JWT เปลี่ยน public API contract — แก้ LP บางส่วนแล้วผ่าน MJ-2 (ownership); TTL/refresh ต้องตัดสินใจ product |
| CODE-SG-3 | ⏸️ เลื่อน | กัน email enumeration ต้องเปลี่ยน behavior ของ register(409)/login(401) ที่ทั้ง spec และ tests ผูกไว้ — ต้องตัดสินใจ product |
| CODE-SG-4 | ⏸️ เลื่อน | Pagination เปลี่ยน response schema + OpenAPI/codegen — design ระบุ "รายการ small — คืนทั้งหมด" (MVP) |

**Gate หลังแก้**: `make test` ผ่าน (coverage 86.1% ≥ 80%) · `go vet ./...` clean · `make build` OK · `gofmt -l` clean

---

## 🔴 Critical

ไม่มี — โครงสร้างสถาปัตยกรรม, dependency direction, การ hash password, การผูก JWT algorithm (HS256 เท่านั้น) และการคืน password hash ล้วนถูกต้อง

---

## 🟡 Major

### CODE-MJ-1 — gRPC รั่ว error ภายในไปยัง client
- **ไฟล์**: `internal/adapter/inbound/grpc/auth.go:84`
- **หมวด**: Error Handling / Security
- **รายละเอียด**: `toServiceError` map error ที่ไม่รู้จัก → `status.Error(codes.Internal, err.Error())` ส่ง raw message (เช่น error จาก mongo driver / infrastructure) กลับไปให้ client ตรงๆ ต่างจากฝั่ง HTTP ที่คืน generic `internal server error` และ log รายละเอียดไว้ฝั่งเซิร์ฟเวอร์
- **ผลกระทบ**: เปิดเผย internal detail (โครงสร้าง infra, ชื่อ collection, classpath) ให้ผู้โจมตีใช้โจมตีต่อ; พฤติกรรมไม่สอดคล้องกันระหว่าง REST กับ gRPC
- **แนวทางแก้**: คืน `status.Error(codes.Internal, "internal server error")` ที่เป็นข้อความคงที่ และ log `err` จริงทางเซิร์ฟเวอร์ (พก `slog` เข้า interceptor หรือ log ที่ handler)

### CODE-MJ-2 — ไม่มี authorization (token ใดอ่าน/แก้/ลบ user ใดก็ได้)
- **ไฟล์**: `internal/adapter/middleware/auth_middleware.go` (ทุก protected route), `internal/adapter/inbound/grpc/auth.go:30-40`
- **หมวด**: Security / Authorization
- **รายละเอียด**: ระบบตรวจแค่ "token ถูกต้อง" เท่านั้น — ทุกคนที่ login สำเร็จ (subject ใดก็ได้ใน token) สามารถ `GET/PUT/DELETE /api/v1/users/:id` ของ user ใดก็ได้ในระบบ เนื่องจาก design ระบุ auth แบบ bearer-only ไม่มี role/ownership จึงถือว่าถูก design แต่สำหรับ deployment สาธารณะ (แม้ multi-tenant ขนาดเล็ก) นี่คือช่องว่างจริง
- **ผลกระทบ**: ผู้ใช้คนหนึ่งแก้/ลบข้อมูลผู้ใช้คนอื่นได้ทั้งหมด
- **แนวทางแก้**: (1) เพิ่ม role เช่น `admin` ใน token claims แล้ว guard routes ที่ต้องการ (2) หรือทำ ownership check: สำหรับ `PUT/DELETE /:id` อนุญาตเฉพาะ subject == id ตาม token จนกว่าจะมี role model

### CODE-MJ-3 — config ไม่ fail-fast เมื่อ secret ว่าง — ลงชื่อ JWT ด้วย secret ว่างได้โดยไม่รู้ตัว
- **ไฟล์**: `config/config.go:129-130` (precedence: `getenv → secret → yaml → ""`), `cmd/api/main.go:53` (`jwt.NewTokenManager(cfg.JWTSecret)`)
- **หมวด**: Security / Correctness
- **รายละเอียด**: ถ้ารันด้วย `APP_ENV=prod` โดยไม่ได้ build `secret_prod.go` (git-ignored) และไม่ได้ตั้ง `JWT_SECRET` env → `cfg.JWTSecret == ""` เซิร์ฟเวอร์ยัง start ปกติและเซ็น HS256 ด้วย **empty secret** (token ปลอมได้ทันที) โดยไม่มี warning ใดๆ ต่อได้เช่นกัน `MongoURI` ว่างก็ log ขึ้นแล้วค่อยตายที่ `db.Connect`
- **ผลกระทบ**: misconfiguration เงียบ → JWT forgeable ใน production
- **แนวทางแก้**: เพิ่มการ validate หลัง `config.Load()` ว่า `MongoURI` และ `JWTSecret` ไม่ว่าง (โดยเฉพาะ env ที่ไม่ใช่ dev) → log + `os.Exit(1)` พร้อมข้อความชัดเจน หรือตรวจใน `config.Load` เอง

---

## 🟢 Minor

### CODE-MN-1 — รูปแบบ error ของ HTTP ต่างจาก api-spec.md
- **ไฟล์**: `internal/adapter/inbound/http/helpers/utils.go:25,54`, test `helpers/utils_test.go:181`; spec `design/api-spec.md:20-27`
- **หมวด**: Design Compliance / Error Handling
- **รายละเอียด**: design เขียน error เป็น object เดียว `{"error": "..."}` แต่ implementation คืน envelope `{"error":"is required","fields":{"name":"is required"}}` — top-level error เป็น fragment ("is required") ที่อ่านไม่รู้เรื่องเมื่ออยู่เดี่ยวๆ
- **แนวทางแก้**: ปรับ api-spec.md ให้เป็น envelope (ตรงกับ AGENTS.md อยู่แล้ว) และเปลี่ยน top-level message เป็น `"validation error: name is required"` (เก็บ `fields` ไว้)

### CODE-MN-2 — spec gRPC ต่างจาก proto จริง
- **ไฟล์**: `design/api-spec.md:192-205` vs `api/proto/user_service.proto:16-20`
- **หมวด**: Design Compliance (docs)
- **รายละเอียด**: design ระบุ wrapper `CreateUserResponse { User user }` / `GetUserResponse` แต่ proto จริง return `User` ตรงๆ
- **แนวทางแก้**: อัปเดต api-spec.md ให้ตรงกับ proto ที่ ship

### CODE-MN-3 — ชื่อ database default ต่างกันระหว่าง spec กับ config
- **ไฟล์**: `design/data-model.md:7` (`default user_mgr`) vs `config/config.go:22` (`defaultMongoDatabase = "userdb"`) + `config/config_dev.yaml:11`
- **หมวด**: Design Compliance (docs)
- **แนวทางแก้**: แก้ data-model.md เป็น `userdb`

### CODE-MN-4 — log MongoURI เต็ม (มี credential) ตอน connect ผิดพลาด
- **ไฟล์**: `cmd/api/main.go:42` (`logger.Error("failed to connect to mongodb", "uri", cfg.MongoURI, ...)`)
- **หมวด**: Security
- **รายละเอียด**: ถ้า URI มี userinfo (เช่น `mongodb://user:pass@host`) จะรั่วลง log/LT → credential leak risk
- **แนวทางแก้**: sanitize ก่อน log (ตัดส่วน credentials ออก หรือ log `cfg.MongoDatabase`/host แทน)

### CODE-MN-5 — auth gRPC ยึด string literal ของ method name
- **ไฟล์**: `internal/adapter/inbound/grpc/auth.go:31,68` (`info.FullMethod == "/userv1.UserService/GetUser"`)
- **หมวด**: Code Quality / Correctness
- **รายละเอียด**: literal ชนิดนี้ถ้าเพิ่ม RPC ใหม่จะถูกตีว่า public โดยปริยาย (security-by-default ผิดทาง); compile ไม่ catch typo
- **แนวทางแก้**: ใช้ constant ที่ protoc สร้างให้ เช่น `userv1_UserService_GetUser_FullMethodName` หรือทำ explicit allowlist (public / protected)

---

## 💡 Suggestion

### CODE-SG-1 — ไม่มี rate limiting ที่ `/auth/login`
- **ไฟล์**: route ใน `internal/adapter/inbound/http/server.go`; design ระบุ "none (MVP — future: per-IP)"
- **ผลกระทบ**: brute-force ต่อ password ของ account ใดก็ได้
- **แนวทาง**: throttling per-IP/IP+email (เช่น fail-before-backoff) ก่อน deploy สาธารณะ

### CODE-SG-2 — JWT 24h เลย ต่ออายุไม่ได้ / ไม่มี revoke
- **ไฟล์**: `internal/adapter/outbound/jwt/jwt.go` (TTL 24h)
- **ผลกระทบ**: ถ้า account ถูกลบ/ปิด token เดิมยังใช้ได้จนหมดอายุ (token subject ไม่ถูก re-verify กับ DB ทุก call)
- **แนวทาง**: TTL สั้นลง + refresh flow, หรือ check subject ยังมีอยู่ (และไม่ถูกปิด) ที่ middleware

### CODE-SG-3 — account enumeration ผ่าน register (409) / login (401)
- **ไฟล์**: `internal/application/services/user/service.go` (Register คืน ErrDuplicateEmail), api-spec mapping 401
- **แนวทาง**: ถ้าเป็นระบบสาธารณะ ใช้ response/payload ที่เท่าเทียม (เช่น login คืนตอบเหมือนกันเสมอ + log ต่างกัน) เพื่อเลี่ยงการตรวจสอบได้ว่าอีเมลมีอยู่ในระบบ

### CODE-SG-4 — `GET /users` คืนรายการไม่จำกัด
- **ไฟล์**: repository list (ไม่มีการ pagination) — ตาม design "รายการ users ขนาดเล็ก — คืนทั้งหมด"
- **แนวทาง**: วาง cursor/page ไว้สำหรับ scale ในอนาคต

---

## Design Compliance Summary

| Design artifact | สถานะ | หมายเหตุ |
|---|---|---|
| `design/components.md` (C1–C12, hexagonal) | ✅ | entry point / ports / adapters / domain ตรง spec |
| `design/data-model.md` | ⚠️ | index `email_1` unique ✅ (EnsureIndexes ที่ startup); default db name ต่าง (MN-3) |
| `design/api-spec.md` | ⚠️ | paths/methods/status codes ✅; error format ต่าง (MN-1), proto wrapper ต่าง (MN-2) |
| `design/testing-strategy.md` | ✅ | โครงสร้างทดสอบ, mock strategy, เกณฑ์ 80% ตรง (ได้ 87.6%) |
| D3 (tech stack, ไม่ใช้ lib ผิดหลัก) | ✅ | stdlib testing, gomock, mtest, bufconn — ไม่มี library เกิน |

## Test Coverage Summary

| พื้นที่ | สถานะ |
|---|---|
| user.Service (fake repo, table-driven) | ✅ ครอบคลุม register/login/get/list/update/delete + sentinel errors |
| statusdb.Service (gomock) | ✅ |
| HTTP handlers (Fiber `app.Test`) | ✅ 201/400/404/409/401/200 |
| gRPC (bufconn) | ✅ success + unauth |
| Mongo adapter (mock collection + opt-in live `TestConnect_Live`) | ✅ |
| JWT / bcrypt / config | ✅ table-driven |
| main composition/shutdown | ⚠️ ยังไม่ถูกทดสอบอัตโนมัติ (Documented: manual) |
| E2E | (ไม่มี frontend — ตาม design) |

## Missing Tests

- main() composition + graceful shutdown (documented แบบ manual) — probing test ที่ 1 file ได้
- ไม่มี test เจาะจงสำหรับ "token ของ user ที่ถูกลบ ยังใช้ได้จน TTL หมด"

## Recommendations

| Priority | Findings |
|---|---|
| **Must fix** | CODE-MJ-1, CODE-MJ-3 |
| **Should fix** (ก่อน public deploy) | CODE-MJ-2, CODE-SG-1, CODE-SG-2 |
| **Consider** | CODE-MN-1..5, CODE-SG-3, CODE-SG-4 |

---

*ตรวจด้วย aidlc-code-review v1.0.0 — อ่าน design docs + source 26 ไฟล์; verify ข้อมูลอ้างอิงกับ repo จริงแล้ว*