# Steering: Product

## Summary
- Product: RESTful user-management API (register, login, CRUD) built in Go
- Users: API consumers / backend integrators (single persona)
- Type: Brownfield · Scope: new (standalone product)

## Overview
A backend coding-challenge deliverable (Path 1: User Management API). Builds a RESTful API in Golang to manage users, using MongoDB for persistence, JWT (HMAC HS256) for authentication, gRPC bonus, and hexagonal (ports & adapters) architecture with clean code practices.

## Problem Statement
The challenge requires a complete, production-quality user-management service: registration + authentication returning signed JWTs, full CRUD on users, MongoDB persistence with proper abstraction, logging middleware, a background concurrency task that counts total users every 10 seconds, and comprehensive unit tests with mocked MongoDB. Evaluation focuses on code quality, structure, readability, security, test coverage, idiomatic Go, and bonus implementations.

## Target Users
- **API consumer / backend integrator** — Calls the shipped HTTP/gRPC endpoints to create, authenticate, list, update, delete users; consumes the prepared documentation (README, JWT guide, sample requests).

## Key Features
- **Auth** — register + login returning JWT (HMAC HS256), Bearer-token middleware protection.
- **User CRUD** — create, get by ID, list, update name/email, delete.
- **MongoDB persistence** — official Go driver, unique email index, interface abstraction (repository ports).
- **Observability** — logging middleware (method, path, status, duration) + structured slog.
- **Concurrency task** — background goroutine every 10 s logging total users via `EstimatedDocumentCount` with graceful shutdown.
- **Bonus** — Docker/docker-compose, input validation, graceful shutdown, hexagonal layering, gRPC (CreateUser/GetUser with token-metadata auth).
- **Requirements-confirmed** (D1 + requirements phase): 16 stories (10 High / 4 Medium / 2 Low) across 5 areas — Auth & Model, User Operations, Persistence, Observability & Operations, Quality; validation requires email format + password ≥ 8; gRPC maps domain errors to NotFound/AlreadyExists/InvalidArgument/Unauthenticated/Internal.

## Domain Language

| Term | Definition | Example |
|------|-----------|---------|
| User | Domain entity: ID, Name, Email (unique), Password (hashed), CreatedAt | `internal/domain/user/entity.go` |
| Port | Interface defining what domain/application needs | `Repository`, `Hasher`, `TokenManager` |
| Adapter | Implementation of a port (inbound/outbound) | MongoDB repo, JWT manager, bcrypt hasher |
| Token | JWT signed HMAC HS256 for authenticated requests | `Authorization: Bearer <jwt>` |
| statusdb | Collection-level stats port (count) | `Count` via `EstimatedDocumentCount` |

## Success Criteria
- All 7 core requirements implemented and passing (`make test`).
- Bonus where valuable: gRPC, Docker, validation, graceful shutdown, hexagonal architecture.
- Deliverables present: README with setup/execution, JWT generation/usage guide, sample requests/responses, assumptions/design decisions docs.
- Metric: test coverage > 80% on non-generated code.

## Constraints & Assumptions
- Constraints: Go standard library for testing; MongoDB for persistence; JWT HMAC HS256 with secret key; 10-second cadence for the concurrency task; code meet evaluation criteria (structure, readability, security, idiomatic Go).
- Assumptions: single-user-role scenario; user emails are the unique business key; document deletions are rare so `EstimatedDocumentCount` accuracy is acceptable for a logging job.

## Existing User Journeys
- **Register**: POST /api/v1/users (public) → validate → hash password → persist → 201 + user JSON.
- **Login**: POST /api/v1/auth/login (public) → verify credentials → return JWT.
- **CRUD**: guarded endpoints (Bearer JWT) for list/get/update/delete users.
- **Pain points**: none observed — journeys implemented and covered by tests; concurrency count job may be slightly under-counted after deletions (accepted tradeoff).

## Project Type
- **Type**: Brownfield (full implementation already present in repo)
- **Scope**: New product (standalone challenge deliverable)