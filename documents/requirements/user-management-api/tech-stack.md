# Tech Stack — User Management API

**Tech Stack: Go 1.26.4 · Fiber v2 · MongoDB (mongo-driver v2) · JWT (HMAC HS256) · bcrypt · oapi-codegen (OpenAPI 3.0.3)**

โปรเจกต์นี้ใช้ **Hexagonal Architecture (Ports & Adapters)** โดยมี **Fiber** เป็น
HTTP framework ฝั่ง inbound adapter

---

## ภาพรวม Architecture

```
┌──────────────────────┐
│     HTTP / REST      │   Fiber + oapi-codegen (OpenAPI 3.0.3)
└──────────┬───────────┘
           │
        Adapter
           │
           ▼
┌────────────────────────────────────────────────────┐
│                  APPLICATION                        │
│                                                    │
│   Use Cases / Application Services                 │
│                                                    │
│   Register()                                       │
│   Login()                                          │
│   List()                                           │
│   GetUser()                                        │
│   UpdateUser()                                     │
│   DeleteUser()                                     │
│   Count()  (statusdb concurrency task)             │
│                                                    │
└───────────────────────┬────────────────────────────┘
                        │
                     Ports
                        │
                        ▼
┌────────────────────────────────────────────────────┐
│                     DOMAIN                          │
│                                                    │
│   Entity                                           │
│   Value Object                                     │
│   Domain Rules                                     │
│   Domain Service                                   │
│                                                    │
│   User / StatusDB                                   │
│                                                    │
└───────────────────────┬────────────────────────────┘
                        │
                     Ports
                        │
                        ▼
┌────────────────────────────────────────────────────┐
│                INFRASTRUCTURE                       │
│                                                    │
│   MongoDB                                          │
│   JWT (HS256)                                      │
│   bcrypt                                           │
│                                                    │
└────────────────────────────────────────────────────┘

Infrastructure → Application → Domain
Adapter        → Application
```

## Project Structure

```
backend-challenge/
│
├── cmd/
│   └── api/
│       └── main.go                  # composition root (Fiber wiring + statusdb task)
│
├── config/                          # top-level config package (YAML + env + secret)
│   ├── config.go                    # config.Load() — YAML/secret lookup + env override
│   ├── config_dev.yaml              # per-environment YAML
│   └── secret_dev.go / secret_prod.go
│
├── internal/
│   │
│   ├── domain/                       # หัวใจของระบบ (ไม่รู้จัก Fiber/HTTP/MongoDB)
│   │   ├── user/
│   │   │   ├── entity.go             # User entity + business rules
│   │   │   ├── errors.go             # sentinel errors
│   │   │   └── repository.go         # port: interface ที่ Domain ต้องการ
│   │   └── statusdb/
│   │       └── repository.go         # port สำหรับ concurrency count task
│   │
│   ├── application/                  # orchestration / use cases
│   │   ├── services/
│   │   │   ├── user/
│   │   │   │   ├── service.go        # Service interface + use cases
│   │   │   │   ├── dto.go            # CreateUserInput, LoginInput, UserOutput, ...
│   │   │   │   └── utils.go          # normalizeEmail, validEmail, ...
│   │   │   └── statusdb/
│   │   │       └── service.go        # Count users (background task)
│   │   └── ports/
│   │       ├── hasher.go             # port สำหรับ password hashing
│   │       └── token.go              # port สำหรับ JWT
│   │
│   ├── adapter/
│   │   ├── middleware/               # shared Fiber middleware
│   │   │   ├── auth_middleware.go    # Bearer JWT auth (+ Authenticate guard)
│   │   │   ├── log_middleware.go     # request logging (method, path, status, duration)
│   │   │   └── recovery_middleware.go# (unused — server uses Fiber built-in recover)
│   │   ├── inbound/
│   │   │   └── http/                 # Fiber handlers (implementation ของ Service port)
│   │   │       ├── server.go         # assembles app + apiHandler (implements ServerInterface)
│   │   │       ├── server_openapi.gen.go  # oapi-codegen output (routes + models)
│   │   │       ├── helpers/utils.go  # DecodeBody + WriteServiceError (shared plumbing)
│   │   │       ├── user/             # Handler + Responder interface (user endpoints)
│   │   │       └── health-check/     # GET /health
│   │   │
│   │   └── outbound/                 # implementation ของ Domain/Application ports
│   │       ├── mongodb/              # user repository (MongoDB)
│   │       ├── jwt/                  # token manager (HMAC HS256)
│   │       └── bcrypt/               # password hasher
│   │
│   ├── infrastructure/
│   │   └── db/mongodb.go             # MongoDB Connect wrapper (+ integration test)
│   │
│   └── mocks/                        # gomock mocks (statusdb, userrepo, ports)
│
├── api/
│   ├── openapi.yaml                  # OpenAPI 3.0.3 spec (source of truth for routes)
│   └── codegen.yaml                  # oapi-codegen configuration
│
├── docs/                             # bundled OpenAPI (swagger.json / swagger.yaml)
├── documents/                        # เอกสาร requirement / design
├── Makefile                          # run / build / test / codegen / mock / docsgen
├── go.mod
└── README.md
```

## Domain Layer

**Domain คือหัวใจของระบบ**

ไม่ควรรู้จัก:
- Fiber
- HTTP
- MongoDB

ตัวอย่าง:

```go
package user

import "time"

type User struct {
    ID        string
    Name      string
    Email     string
    Password  string // hashed
    CreatedAt time.Time
}
```

Business rule อยู่ตรงนี้:

```
User
 ├── ตรวจ email format
 ├── normalize email (lowercase)
 └── Business Rules
```

## Port

**Port คือ interface ที่บอกว่า Domain/Application ต้องการอะไร**

ตัวอย่าง Repository:

```go
package user

import "context"

type Repository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    List(ctx context.Context) ([]User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

สังเกตว่า Domain ไม่สนใจว่าเก็บที่ไหน

```
Repository
    │
    ├── MySQL
    ├── PostgreSQL
    ├── MongoDB   ← เราเลือก MongoDB
    └── Mock      ← ใช้ใน unit test
```

Domain เห็นแค่ Repository

## Application Layer

**Application ทำหน้าที่ Orchestration / Use Case**

ตัวอย่าง `Register`:

1. Validate input
2. Check duplicate email
3. Hash password
4. Save User
5. Return UserOutput

```go
func (s *service) Register(ctx context.Context, in CreateUserInput) (UserOutput, error) {
    name := strings.TrimSpace(in.Name)
    email := normalizeEmail(in.Email)

    if name == "" {
        return UserOutput{}, &domain.ValidationError{Field: "name", Message: "is required"}
    }
    if !validEmail(email) {
        return UserOutput{}, &domain.ValidationError{Field: "email", Message: "is invalid"}
    }
    if len(in.Password) < passwordMinLength {
        return UserOutput{}, &domain.ValidationError{Field: "password", Message: "must be at least 8 characters"}
    }

    if _, err := s.repo.FindByEmail(ctx, email); err == nil {
        return UserOutput{}, domain.ErrDuplicateEmail
    } else if !errors.Is(err, domain.ErrNotFound) {
        return UserOutput{}, err
    }

    hashed, err := s.hasher.Hash(in.Password)
    if err != nil {
        return UserOutput{}, err
    }

    u := &domain.User{Name: name, Email: email, Password: hashed, CreatedAt: time.Now().UTC()}
    if err := s.repo.Create(ctx, u); err != nil {
        return UserOutput{}, err
    }
    return toOutput(u), nil
}
```

`Login` อีกฝั่ง ก็เรียก port เดียวกัน: `repo.FindByEmail` → `hasher.Compare` → `tokens.Generate(id)`

## Outbound Adapter

**ตัวนี้คือ implementation ของ Port**

เช่น Domain บอกว่า:

```go
type Repository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
}
```

ดังนั้น:

```
Domain Port
     │
     │ Repository
     ▼
MongoDB Adapter
     │
     ▼
MongoDB
```

Package: `internal/adapter/outbound/mongodb/user_repository.go`

## Inbound Adapter

**เช่น HTTP API**

Routes ถูก **gen เป็น Go code จาก `api/openapi.yaml`** ด้วย oapi-codegen
(run ผ่าน `make codegen` → `internal/adapter/inbound/http/server_openapi.gen.go`):

```
POST   /api/v1/users          (public)
POST   /api/v1/auth/login     (public)
GET    /health                (public)
GET    /api/v1/users          (protected)
GET    /api/v1/users/:id      (protected)
PUT    /api/v1/users/:id      (protected)
DELETE /api/v1/users/:id      (protected)
```

Fiber handler (`internal/adapter/inbound/http/user/handler.go`):

```go
// Handler wires the user service to Fiber handlers.
type Handler struct {
    Users appuser.Service
    Responder // DecodeBody + WriteServiceError
}

// Register handles POST /api/v1/users.
func (h *Handler) Register(c *fiber.Ctx) error {
    var in appuser.CreateUserInput

    if !h.DecodeBody(c, &in) {
        return nil
    }

    out, err := h.Users.Register(c.UserContext(), in)
    if err != nil {
        return h.WriteServiceError(c, err)
    }

    return c.Status(fiber.StatusCreated).JSON(out)
}
```

Route mapping ระหว่าง Fiber ↔ use case อยู่ที่ `server.go` (`apiHandler`
implement `ServerInterface` ของ oapi-codegen) และ guarded operation ใช้
`middleware.Authenticate` ตรวจ Bearer token ก่อนเข้า use case:

```
HTTP
 │
 ▼
Fiber + oapi-codegen (server_openapi.gen.go)
 │
 ▼
apiHandler.guard → middleware.Authenticate (JWT)
 │
 ▼
user.Handler  (ใช้ Responder ทำ decode/encode)
 │
 ▼
Use Case  (user.Service)
 │
 ▼
Domain
```

**ถ้าวันหนึ่งเปลี่ยน Fiber → Gin (หรือในทางกลับกัน) Domain และ Application แทบไม่ต้องเปลี่ยน**

นี่คือจุดที่รับผิดชอบของ **Adapter/Infrastructure** เท่านั้น

## Dependency Injection

**ทั้งหมดจะถูกประกอบที่ main.go (composition root)**

ตัวอย่าง (ปัจจุบัน = `cmd/api/main.go`):

```go
func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    slog.SetDefault(logger)

    cfg, err := config.Load() // YAML + secret file + env override
    if err != nil { /* exit */ }

    client, err := db.Connect(ctx, cfg.MongoURI) // internal/infrastructure/db
    if err != nil { /* exit */ }
    defer client.Disconnect(context.WithoutCancel(ctx))

    userRepo := outmongodb.NewUserRepository(client.Database(cfg.MongoDatabase).Collection("users"))
    userRepo.EnsureIndexes(ctx) // unique email index

    tokenManager := jwt.NewTokenManager(cfg.JWTSecret)
    userService := user.NewService(userRepo, bcrypt.New(), tokenManager)

    go countUsersLoop(ctx, statusdb.NewService(userRepo), logger) // concurrency task

    app := httpapi.NewServer(userService, tokenManager, logger)
    app.Listen(":" + cfg.Port)
}
```

**นี่คือจุดที่ Infrastructure ถูกเอามาประกอบเข้ากับ Application**

## Tooling / Build

การ build, test และ code generation จัดการผ่าน `Makefile`:

```
make build            # go build → bin/api
make run              # go run ./cmd/api  (APP_ENV=prod สำหรับ production)
make test             # go test ./... + coverage (ไม่นับ generated/mocks)
make test-integration # MongoDB integration test (ต้องตั้ง TEST_MONGO_URI)
make codegen          # oapi-codegen: api/openapi.yaml → server_openapi.gen.go
make mock             # mockgen: domain/application ports → internal/mocks/
make docsgen          # bundle api/openapi.yaml → docs/ (swagger.json / swagger.yaml)
```

## Dependency Direction

สิ่งที่สำคัญที่สุดของ Hexagonal Architecture คือ:

```
              Inbound
             Adapters
                │
                ▼
        ┌────────────────┐
        │  Application   │
        └───────┬────────┘
                │
                ▼
        ┌────────────────┐
        │     Domain     │
        └────────────────┘
                ▲
                │
        ┌───────┴────────┐
        │     Ports      │
        └───────┬────────┘
                ▲
                │
        ┌───────┴────────┐
        │    Outbound    │
        │    Adapters    │
        └────────────────┘
```

หรือจำง่าย ๆ:

> **Domain ไม่รู้จักโลกภายนอก แต่โลกภายนอกต้องรู้จัก Domain**

## สรุป (ไว้ตอบ Interview)

> Hexagonal Architecture separates the core business logic from external
> concerns by defining ports as interfaces and implementing them through
> adapters. The domain contains business rules, the application layer
> orchestrates use cases, while infrastructure adapters handle databases,
> messaging systems, HTTP frameworks, and external services. This reduces
> coupling and makes the system easier to test, maintain, and replace external
> technologies.

สำหรับ Go เรามองว่าโครงสร้างที่ดีคือ:

```
Domain
  ↓
Application
  ↓
Ports
  ↓
Adapters
  ↓
Infrastructure
```