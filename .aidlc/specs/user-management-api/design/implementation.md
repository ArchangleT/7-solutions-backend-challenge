# Implementation — User Management API

Engineer's reference สำหรับ scaffold และ daily development

## Code Organization

- **Architecture pattern**: Hexagonal (Ports & Adapters) — dependency: `adapter → application → domain`; outbound adapters implement domain/app ports
- **Repository type**: Single repo (หนึ่ง module `github.com/ArchangleT/7-solutions-backend-challenge`)

### Directory structure

```
backend-challenge/
├── cmd/api/main.go                     # composition root (Fiber + gRPC + statusdb loop + graceful shutdown)
├── config/
│   ├── config.go                       # config.Load() — YAML/secret + env override
│   ├── config_dev.yaml                 # per-environment YAML
│   ├── secret_dev.go / secret_prod.go  # build-tagged secret values
├── api/
│   ├── openapi.yaml                    # OpenAPI 3.0.3 (source of truth REST)
│   ├── codegen.yaml                    # oapi-codegen config
│   └── proto/user_service.proto        # gRPC spec
├── internal/
│   ├── domain/
│   │   ├── user/                       # entity.go, errors.go, repository.go (port)
│   │   └── statusdb/                   # repository.go (port: Count)
│   ├── application/
│   │   ├── ports/                      # hasher.go, token.go
│   │   └── services/
│   │       ├── user/                   # service.go, dto.go, utils.go
│   │       └── statusdb/               # service.go
│   ├── adapter/
│   │   ├── middleware/                 # auth_middleware.go, log_middleware.go, recovery_middleware.go (unused)
│   │   ├── inbound/
│   │   │   ├── http/                   # server.go, server_openapi.gen.go, user/, health-check/, helpers/
│   │   │   └── grpc/                   # server.go, auth.go, userv1/*.pb.go
│   │   └── outbound/
│   │       ├── mongodb/                # user_repository.go
│   │       ├── jwt/                    # jwt.go
│   │       └── bcrypt/                 # bcrypt.go
│   ├── infrastructure/db/              # mongodb.go (Connect wrapper)
│   └── mocks/                          # gomock (statusdb, userrepo, ports)
├── docs/                               # bundled swagger.json / swagger.yaml
├── Makefile · Dockerfile · docker-compose.yml · go.mod
```

### Module boundaries
- `domain`: ไม่รู้จัก Fiber/HTTP/MongoDB/gRPC — แค่ entity, errors, port interfaces
- `application`: orchestration ผ่าน ports เท่านั้น
- `adapter/inbound`: การรับ request (Fiber handlers, gRPC server) + middleware
- `adapter/outbound`: implementation ของ ports (mongodb/jwt/bcrypt)
- `infrastructure/db`: MongoDB connect wrapper

### Naming conventions
- package: ตาม Go idiom, ตั้งสั้น (เช่น `user`, `statusdb`, `jwt`)
- port file: `repository.go`, `token.go`, `hasher.go`
- generated: `server_openapi.gen.go`, `*.pb.go`, `internal/mocks/`

## Technology Stack

| Dependency | Version |
|------------|---------|
| `github.com/gofiber/fiber/v2` | v2.52.15 |
| `go.mongodb.org/mongo-driver/v2` | v2.9.1 |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 |
| `github.com/oapi-codegen/runtime` | v1.7.0 |
| `golang.org/x/crypto` | v0.57.0 (bcrypt) |
| `google.golang.org/grpc` | v1.83.2 |
| `google.golang.org/protobuf` | v1.36.11 |
| `go.uber.org/mock` | v0.6.0 (test) |
| `github.com/getkin/kin-openapi` | v0.149.0 (validation) |
| `gopkg.in/yaml.v3` | v3.0.1 |
| Go toolchain | 1.26.4 |

Generated toolchain: `oapi-codegen` (codegen), `mockgen` (mock), `protoc-gen-go` (+ `protoc-gen-go-grpc`) (proto)

## Development Setup

### Prerequisites
- Go 1.26.4+
- (ไม่บังคับ) Docker + Docker Compose สำหรับรัน MongoDB/API
- (กรณี regen เฉพาะ) `make tools` installs oapi-codegen, mockgen, protoc plugins

### Setup commands (จาก fresh clone)
```bash
make test          # รันทั้งหมด + coverage (ไม่นับ generated/mocks)
make run           # go run ./cmd/api  (dev config)
docker-compose up  # ขึ้น MongoDB + API + healthcheck
```

### Env variables

| Variable | Description | Example |
|----------|-------------|---------|
| `APP_ENV` | environment (dev/prod) | `prod` |
| `PORT` | HTTP listen port | `8080` |
| `GRPC_PORT` | gRPC listen port | `50051` |
| `TEST_MONGO_URI` | live MongoDB URI (เฉพาะ integration test) | `mongodb://localhost:27017` |
| JWT secret | ผ่าน config/secret file per env | `<secret>` (แก้ใน `config/secret_*.go`) |

*(ตัวอย่างใช้ placeholder — ไม่ใช่ secret จริง)*

## Testing

| Level | Framework / approach | Run |
|-------|----------------------|-----|
| Unit (service/handlers/adapters/utils) | Go stdlib `testing` + table-driven (+ gomock) | `go test ./...` |
| HTTP handler | Fiber `app.Test` (in-process, ไม่ต้อง bind port) | `go test ./internal/adapter/inbound/http/...` |
| gRPC | bufconn in-process + fake service | `go test ./internal/adapter/inbound/grpc/...` |
| MongoDB live | opt-in (ต้อง `TEST_MONGO_URI`) | `make test-integration` |
| Coverage | `go test -coverprofile` (exclude `internal/mocks/`, `server_openapi.gen.go`, `*.pb.go`) | `make test` |

- **Coverage target**: ≥ 80% บน non-generated code
- เอกสารรายละเอียดเพิ่มเติมที่ `design/testing-strategy.md`