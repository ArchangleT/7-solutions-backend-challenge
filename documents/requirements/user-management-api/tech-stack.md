# Tech Stack — User Management API

**Tech Stack: Go 1.26 · Fiber v2 · MongoDB · JWT (HMAC HS256) · bcrypt**

โปรเจกต์นี้ใช้ **Hexagonal Architecture (Ports & Adapters)** โดยมี **Fiber** เป็น
HTTP framework ฝั่ง inbound adapter

---

## ภาพรวม Architecture

```
┌──────────────────────┐
│     HTTP / REST      │   Fiber (เดิม Gin / Fiber / Echo)
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
│   GetUser()                                        │
│   UpdateUser()                                     │
│   DeleteUser()                                     │
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
│   User                                              │
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
│       └── main.go                  # composition root (Fiber wiring)
│
├── internal/
│   │
│   ├── domain/                       # หัวใจของระบบ (ไม่รู้จัก Fiber/HTTP/MongoDB)
│   │   └── user/
│   │       ├── entity.go             # User entity + business rules
│   │       ├── errors.go             # sentinel errors
│   │       └── repository.go         # port: interface ที่ Domain ต้องการ
│   │
│   ├── application/                  # orchestration / use cases
│   │   ├── user/
│   │   │   ├── service.go            # Service interface + use cases
│   │   │   └── dto.go
│   │   └── ports/
│   │       ├── hasher.go             # port สำหรับ password hasing
│   │       └── token.go              # port สำหรับ JWT
│   │
│   ├── adapter/
│   │   ├── inbound/
│   │   │   └── http/                 # Fibber handlers (implementation ของ Service port)
│   │   │       ├── server.go
│   │   │       ├── handler.go
│   │   │       ├── middleware.go
│   │   │       └── response.go
│   │   │
│   │   └── outbound/                 # implementation ของ Domain/Application ports
│   │       ├── mongodb/              # user repository (MongoDB)
│   │       ├── jwt/                  # token manager (HMAC HS256)
│   │       └── bcrypt/               # password hasher
│   │
│   └── infrastructure/
│       └── config/                   # การตั้งค่า (env)
│
├── documents/                        # เอกสาร requirement / design
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

type User struct {
    ID         string    `bson:"_id" json:"id"`
    Name       string    `json:"name"`
    Email      string    `json:"email"`
    Password   string    `json:"-"` // hashed
    CreatedAt  time.Time `json:"created_at"`
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

type Repository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    List(ctx context.Context) ([]User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
    Count(ctx context.Context) (int64, error)
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
    }

    hashed, _ := s.hasher.Hash(in.Password)
    u := &domain.User{Name: name, Email: email, Password: hashed, CreatedAt: time.Now().UTC()}
    if err := s.repo.Create(ctx, u); err != nil {
        return UserOutput{}, err
    }
    return toOutput(u), nil
}
```

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

```
POST /api/v1/users
```

Handler (Fiber):

```go
func (s *Server) register(c *fiber.Ctx) error {
    var in user.CreateUserInput

    if !s.decodeBody(c, &in) {
        return nil
    }

    out, err := s.users.Register(c.UserContext(), in)
    if err != nil {
        return s.writeServiceError(c, err)
    }

    return c.Status(fiber.StatusCreated).JSON(out)
}
```

```
HTTP
 │
 ▼
Fiber Handler
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

ตัวอย่าง:

```go
func main() {
    cfg := config.Load()

    client := mongodb.Connect(ctx, cfg.MongoURI)
    userRepo := mongodb.NewUserRepository(client.Database(cfg.MongoDatabase).Collection("users"))

    userService := user.NewService(userRepo, bcrypt.New(), jwt.NewTokenManager(cfg.JWTSecret))

    app := httpapi.NewServer(userService, jwt.NewTokenManager(cfg.JWTSecret), logger)

    app.Listen(":" + cfg.Port)
}
```

**นี่คือจุดที่ Infrastructure ถูกเอามาประกอบเข้ากับ Application**

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