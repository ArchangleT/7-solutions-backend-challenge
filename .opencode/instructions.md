# Project Instructions

## Tech Stack

- Languages and frameworks: Go (module `github.com/7-solutions/backend-challenge`, go 1.26.4)
- Database: MongoDB (official Go driver)
- Authentication: JWT (HS256 / HMAC, secret-key signed)
- Testing: Go standard `testing` package, MongoDB interactions mocked
- Optional bonus: Docker + docker-compose, gRPC (with `.proto`), hexagonal architecture (ports and adapters)
- OpenCode config uses the `serena` MCP server (`.opencode/opencode.json`)

## Commands

- Build: `go build ./...`
- Test: `go test ./...`
- Run: `go run ./cmd/...` (entry point chosen during implementation; see README for setup)
- Dependency management: `go get` / `go mod tidy`

## Conventions

- Idiomatic Go: use interfaces to abstract MongoDB operations for testability
- Clean code + hexagonal architecture: separate domain, application, and infrastructure layers; decouple business logic from frameworks
- Keep HTTP handlers thin; route through service/repository layers
- Passwords must be hashed before persistence
- Unit tests based on standard library; mock MongoDB where appropriate
- Do not write code for the Lottery Search System — it is a design-only deliverable (design document)

## Architecture

- Part 1 — User Management API:
  - REST API server (HTTP)
  - Authentication layer: registration, login returning JWT; JWT validation middleware
  - User operations: create, get by ID, list all, update name/email, delete
  - Persistence: MongoDB through an interface (repository)
  - Cross-cutting: logging middleware (method, path, execution time); background goroutine counting users every 10 seconds; graceful shutdown via `context.Context` and signal handling
  - Optional: gRPC server (`CreateUser`, `GetUser`) secured with token metadata
- Part 2 — Lottery Search System:
  - Design-only exercise, no code
  - 10M+ six-digit tickets; 6-char search pattern with digit and `*` wildcards
  - Must address: indexing/algorithm for wildcard matching, atomic non-duplicate allocation across concurrent users, recommended production storage