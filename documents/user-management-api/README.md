# User Management API — Setup & Design

A RESTful user management API in Go using **Fiber v2**, **MongoDB**, **JWT (HS256)**
for authentication, and **bcrypt** for password hashing, structured with
**Hexagonal Architecture** (Ports & Adapters).

---

## Architecture

The project follows hexagonal (ports & adapters) architecture: business logic
lives in the inner core and depends only on interfaces (ports), while concrete
technologies (Fiber, MongoDB, JWT library, bcrypt) are supplied by outward
adapters.

```
                        ┌──────────────────────────────────────┐
                        │          cmd/api (main.go)           │  composition root
                        └──────────────────┬───────────────────┘
                                           │
          inbound adapters (driving)       │         outbound adapters (driven)
     ┌─────────────────────────────┐       │      ┌─────────────────────────────┐
     │  internal/adapter/          │       │      │  internal/adapter/          │
     │    inbound/http/            │       │      │    outbound/mongodb/        │
     │  Fiber handlers, middleware ├───────┼─────►│  UserRepository + connect   │
     │  + router (app.Listen)      │       │      └─────────────────────────────┘
     └─────────────────────────────┘       │
          route: Fiber app                 │      ┌─────────────────────────────┐
     ┌─────────────────────────────┐       │      │  outbound/jwt/              │
     │  middleware: requireAuth    │       ├─────►│  TokenManager (HS256)       │
     │  + c.Locals("user_id")      │       │      └─────────────────────────────┘
     └─────────────────────────────┘       │      ┌─────────────────────────────┐
                                           ├─────►│  outbound/bcrypt/           │
                                           │      │  PasswordHasher (bcrypt)    │
                                           │      └─────────────────────────────┘
                                           ▼
     ┌────────────────────────────────────────────────────────────┐
     │              internal/ (hexagonal layout)                  │
     │                                                            │
     │   domain/user/       pure entity + errors + repository port│
     │   application/       use cases (user.Service)              │
     │     ports/           interfaces: PasswordHasher, TokenMgr  │
     └────────────────────────────────────────────────────────────┘
```

### Layering rules

- **Domain** (`internal/domain/user/`) — the `User` entity, sentinel errors, and
  the `Repository` port. Imports nothing from the standard library beyond basics.
  Zero framework knowledge.
- **Application** (`internal/application/`) — the use cases in
  `services/user/`, the `user` DTOs, and port interfaces (`application/ports/`:
  `ports.PasswordHasher`, `ports.TokenManager`). Depends on the domain and its
  own ports, never on a driver or web framework.
- **Adapters** (`internal/adapter/`) — concrete implementations of the ports:
  `outbound/mongodb/` (MongoDB repository), `outbound/bcrypt/` (password hasher),
  `outbound/jwt/` (token manager), and the inbound `inbound/http/` layer
  (Fiber handlers consuming `user.Service`). Shared middleware lives in
  `internal/adapter/middleware/` (`auth_middleware.go`).
- **Infrastructure** — cross-cutting concerns live in top-level and
  `internal/infrastructure/` packages: `config/` for environment-based
  configuration and `internal/infrastructure/db/` for the MongoDB connection.
- **Composition root** (`cmd/api/main.go`) — wires adapters into ports and
  starts the Fiber server with graceful shutdown.

### Project layout

```
cmd/
  api/main.go                          composition root + Fiber + graceful shutdown
config/                                 YAML config loader + secret_{env}.go (top-level)
  config_dev.yaml                      dev environment defaults
  secret_dev.go                        dev secrets
  secret_prod.go                       production secrets (gitignored; set via APP_ENV=prod)
internal/
  domain/
    user/                              User entity + errors + repository port
    statusdb/                          statusdb.Repository port (Count)
  application/
    ports/                             PasswordHasher, TokenManager interfaces
    services/
      user/                            user.Service interface + use cases + DTOs + utility helpers
      statusdb/                        statusdb.Service (count users task)
  infrastructure/
    db/mongodb.go                    MongoDB Connect wrapper (+ integration test)
  adapter/
    middleware/
      auth_middleware.go              shared auth + logging middleware (Fiber)
    inbound/
      http/
        server.go                      Fiber app + compose handlers
        server_openapi.gen.go          oapi-codegen output (ServerInterface + models)
        utility/utils.go               DecodeBody + WriteServiceError (shared plumbing)
        user/                          Responder interface + Handler (user endpoints)
        healthcheck/                   Health endpoint
    outbound/
      mongodb/                         UserRepository (+ integration test)
      jwt/                             JWT HS256 token manager + tests
      bcrypt/                          bcrypt password hasher + tests
  mocks/                               gomock mocks (statusdb, userrepo, ports)
docs/                                   OpenAPI bundle (make docsgen)
api/
  openapi.yaml                        OpenAPI 3.0.3 spec
  codegen.yaml                        oapi-codegen configuration
Makefile                               run, test, mock, codegen, docsgen, tools, help
```

---

## Prerequisites

- Go 1.26+
- MongoDB 5+ (local or via Docker)
- Docker (optional, for `docker compose`)

## Running locally

```bash
# 1. start MongoDB
docker compose up -d mongo

# 2. run the API
go run ./cmd/api
```

Defaults: API on `:8080`, MongoDB at `mongodb://localhost:27017`, database `userdb`.

### Configuration

Config is loaded from `config/config_<app_env>.yaml`, overlaid with secrets from
`config/secret_<app_env>.go`, then overridden by environment variables.
`APP_ENV` selects the environment (defaults to `dev`).

| Setting          | Env var         | Dev default                    | Description                 |
| ---------------- | --------------- | ------------------------------ | --------------------------- |
| Environment      | `APP_ENV`       | `dev`                          | selects `config_<env>.yaml` |
| Port             | `PORT`          | `8080`                         | HTTP listen port            |
| Mongo URI        | `MONGO_URI`     | `mongodb://localhost:27017`    | MongoDB connection string   |
| Mongo database   | `MONGO_DATABASE`| `userdb`                       | database name               |
| JWT secret       | `JWT_SECRET`    | `dev-only-jwt-secret` (dev)    | HMAC secret (production is a gitignored injected secret) |

## Running with Docker Compose

```bash
docker compose up --build
```

Runs the API and MongoDB together. The API listens on `http://localhost:8080`.

---

## Testing

```bash
go test ./...
go vet ./...
make test                            # runs tests + prints coverage report
```

Tests are pure unit tests (no live MongoDB required):

| Package | What is tested |
| ------- | -------------- |
| `config/` | YAML loading, env overrides, secret registry, load errors |
| `internal/infrastructure/db/` | Mongo URI parsing, ping timeout, offline happy path (stubbed) |
| `internal/application/services/user/` | Use cases against in-memory fakes |
| `internal/application/services/statusdb/` | Count use case against generated gomock mock |
| `internal/adapter/inbound/http/user/` | REST endpoints against a fake service via `app.Test()` |
| `internal/adapter/inbound/http/healthcheck/` | Health endpoint |
| `internal/adapter/outbound/jwt/` | Token generation, validation, expiry, tamper detection |
| `internal/adapter/outbound/bcrypt/` | Hash & compare behaviour |

### Integration tests (opt-in)

Live MongoDB connectivity and repository ↔ MongoDB translation are covered
by opt-in integration tests when `TEST_MONGO_URI` is set:

```bash
docker compose up -d mongo
TEST_MONGO_URI=mongodb://localhost:27017 make test-integration
TEST_MONGO_URI=mongodb://localhost:27017 go test ./internal/adapter/outbound/mongodb/ -run Integration -v
```

### Code generation

```bash
make tools                            # install oapi-codegen + mockgen
make codegen                          # regenerate server_openapi.gen.go from api/openapi.yaml
make mock                             # regenerate internal/mocks/ from source interfaces
make docsgen                          # bundle api/openapi.yaml → docs/
```

---

## API reference

| Method  | Path                  | Auth     | Description                  |
| ------- | --------------------- | -------- | ---------------------------- |
| `GET`   | `/health`             | No       | health check                 |
| `POST`  | `/api/v1/users`       | No       | register a user              |
| `POST`  | `/api/v1/auth/login`  | No       | authenticate, returns JWT    |
| `GET`   | `/api/v1/users`       | Bearer   | list all users               |
| `GET`   | `/api/v1/users/:id`   | Bearer   | fetch a user by id           |
| `PUT`   | `/api/v1/users/:id`   | Bearer   | update name and/or email     |
| `DELETE` | `/api/v1/users/:id`  | Bearer   | delete a user                |

All protected endpoints require a valid JWT `Authorization: Bearer <token>` header.

## JWT usage guide

1. Register or log in to obtain a token:

   ```bash
   curl -s -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"alice@example.com","password":"secret123"}'
   ```

   Response:

   ```json
   {
     "token": "<your-jwt>",
     "user": {
       "id": "66f8...",
       "name": "Alice",
       "email": "alice@example.com",
       "created_at": "2026-09-12T07:00:00Z"
     }
   }
   ```

2. Send the token as a Bearer header on protected endpoints:

   ```bash
   curl http://localhost:8080/api/v1/users \
     -H "Authorization: Bearer <your-jwt>"
   ```

   Tokens are signed with HMAC `HS256` and expire after 24 hours. In a debugger
   you can inspect the payload:

   ```bash
   echo "<your-jwt>" | cut -d. -f2 | base64 -d
   ```

## Sample requests

```bash
# register
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"secret123"}'

# get a user by id (with token)
curl http://localhost:8080/api/v1/users/66f8... \
  -H "Authorization: Bearer <your-jwt>"

# update name/email (partial)
curl -X PUT http://localhost:8080/api/v1/users/66f8... \
  -H "Authorization: Bearer <your-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"email":"alice.cooper@example.com"}'

# list all users
curl http://localhost:8080/api/v1/users -H "Authorization: Bearer <your-jwt>"

# delete a user
curl -X DELETE -i http://localhost:8080/api/v1/users/66f8... \
  -H "Authorization: Bearer <your-jwt>"
```

---

## Concurrency task

On startup a background goroutine logs the total number of users in the
database every 10 seconds:

```
time=... msg=user count task started
time=... msg=total users count=104203
time=... msg=total users count=104215
```

It uses `EstimatedDocumentCount` (fast, metadata-based). It stops cleanly when
the server receives `SIGINT`/`SIGTERM` — see `countUsersLoop` in
`cmd/api/main.go`.

---

## Assumptions & design decisions

- **Fiber v2** as the HTTP framework — lightweight, fast (`fasthttp`-backed),
  native context support via `*fiber.Ctx`. Routing uses `/:param` path
  parameters; auth middleware stores the user ID via `c.Locals()`.
- **Passwords** are stored as bcrypt hashes only; the hash never leaves the
  application layer (password-less `UserOutput` DTO).
- **Email** is normalized to lowercase and validated with `net/mail`; the unique
  index on `email` enforces uniqueness at the database level too (a race-proof
  backstop on top of the application-level check).
- **IDs** are MongoDB `ObjectID` hex strings surfaced through the API as opaque
  strings; the domain never imports BSON types.
- **Update** endpoint supports partial updates (name, email, or both). Setting an
  email that already belongs to another user returns `409 Conflict`.
- **Error mapping** is centralized in the HTTP adapter: domain sentinel errors
  are translated to idiomatic HTTP status codes (`404`/`409`/`422`/`401`/`500`)
  so the application layer stays HTTP-agnostic.
- **Token manager** validates issuer, expiry and algorithm (HS256 only); a wrong
  secret or tampered token is rejected by the middleware.
- **Configuration** — per-environment YAML files under `config/`, overlaid with
  Go-registered secrets and overridable via environment variables, so the same
  binary works in dev and production (`APP_ENV=prod` must inject `JWT_SECRET`).
- **Body validation** — unknown JSON fields are rejected (`DisallowUnknownFields`),
  oversized bodies are capped at 1 MiB (`bodyLimit`), and a single JSON object
  is enforced (no arrays at top level).
- **Graceful shutdown** — on `SIGINT`/`SIGTERM` the context is cancelled,
  `app.Shutdown()` drains in-flight requests, the MongoDB client disconnects,
  and the user count goroutine exits — all before the process terminates.
