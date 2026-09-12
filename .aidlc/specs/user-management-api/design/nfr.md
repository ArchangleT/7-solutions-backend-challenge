# Non-Functional Requirements & Infrastructure

---

## Performance

- **Response Time**: เป้าหมาย < 200ms p95 สำหรับ REST endpoints (local/containerized; ข้อมูล user ขนาดเล็ก)
- **Throughput**: ไม่มี target ทางการ (MVP — evaluation จับที่ correctness มากกว่า load)
- **Concurrent Users**: ไม่มี target สูง — background count task (ทุก 10 วิ) ใช้ `EstimatedDocumentCount` เพื่อไม่ให้ scan collection

---

## Scalability

- **Horizontal Scaling**: ไม่บังคับ — single instance; ถ้าต้อง scale เพิ่ม instance ได้ (stateless API, shared MongoDB)
- **Data Growth**: collection `users` เติบโตตามจำนวน register; count task ทนต่อ growth เพราะใช้ metadata count (O(1))

---

## Security

- **Authentication**: JWT (HMAC HS256, secret key จาก config/env) — bearer middleware สำหรับ guarded endpoints; gRPC ใช้ token metadata interceptor
- **Authorization**: none (single role — ทุก authed user เข้าถึงเหมือนกัน; ไม่มี RBAC)
- **Encryption**: At rest — ตาม platform (MongoDB) ที่ deploy; In transit — TLS ถ้า deploy ผ่าน reverse proxy (ไม่ได้ configure ใน compose นี้)
- **Compliance**: none (deliverable challenge — ไม่ใช่ production data)
- **Secret Management**: JWT secret ผ่าน `config/secret_*.go` (build-tagged per env) + env override; ไม่ commit secret จริง

**Roles & Permissions**:
| Role | Permissions |
|------|-------------|
| (ทุก authed user) | List/Get/Update/Delete users — ไม่มี role hierarchy |

---

## Availability & Reliability

- **Uptime SLA**: best effort (challenge deliverable)
- **RPO/RTO**: none (ไม่ใช่ production)
- **Backup**: none จัดการ (MongoDB ท้องถิ่น)
- **Health Check**: `GET /health` endpoint + docker-compose healthcheck ปกติ
- **Graceful shutdown**: `signal.NotifyContext(SIGINT/SIGTERM)` → `app.Shutdown()`, `grpcSrv.GracefulStop()`, `client.Disconnect()` + หยุด count loop

---

## Infrastructure

### Compute
| Component | Service | Configuration |
|-----------|---------|---------------|
| API | Docker container (multi-stage build) | จาก `Dockerfile`; runtime `bin/api_user_mgr` |

### Storage
| Data Type | Service | Configuration |
|-----------|---------|---------------|
| users | MongoDB (mongo:7) | collection `users` + unique index `email_1` |

### Networking
| Component | Service | Configuration |
|-----------|---------|---------------|
| API (HTTP) | port `PORT` (default 8080) | exposed ผ่าน docker-compose |
| API (gRPC) | port `GRPC_PORT` (default 50051) | exposed ผ่าน docker-compose |
| MongoDB | port 27017 | internal (compose network) |

---

## Caching

| Data | Technology | TTL | Invalidation |
|------|-----------|-----|--------------|
| none | — (D3-8: ไม่ใช้ caching ใน MVP) | — | — |

---

## Data Management

- **Migration Tool**: none จำเป็น — MongoDB schema-less; unique index `EnsureIndexes` ตอน startup
- **Retention**: ไม่มี policy พิเศษ (ข้อมูล challenge)
- **Archival**: none
- **Deletion**: hard delete (`DeleteOne`)

---

## Observability

| Concern | Tool | Configuration |
|---------|------|---------------|
| Logging | `log/slog` (text handler) + logging middleware (method/path/status/duration) | stdout |
| Metrics | none (D3-13: structured logging only) | — |
| Tracing | none | — |
| Alerting | none | — |
| Background task | `countUsersLoop` log "total users" ทุก 10 วิ | slog.Info |

---

## CI/CD & Deployment

- **Deploy Strategy**: manual/local — `docker-compose up` (api + mongo:7 + healthcheck)
- **Rollback**: re-pull/rebuild image — ไม่มี pipeline อัตโนมัติ
- **Quality gates**: `make test` (unit + coverage ≥ 80% non-generated), `make build`

| Environment | Trigger | URL |
|-------------|---------|-----|
| Development | local `make run` | `http://localhost:8080` |
| Production | `APP_ENV=prod` (config/secret_prod) | `http://localhost:8080` |