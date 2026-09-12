# Deployment Decisions — user-management-api

## Context Summary
- **Stack**: Go 1.26.4 / Fiber v2 / MongoDB (mongo-driver v2) / JWT (HMAC HS256) / gRPC / bcrypt
- **Architecture**: Hexagonal (Ports & Adapters) — domain / application / adapter / infrastructure
- **Feature**: RESTful API จัดการผู้ใช้ ด้วย Go, MongoDB และ JWT authentication
- **Impact**: New standalone
- **Complexity**: Medium — 13 stories (7 core + 6 bonus), 1 domain, 1 user type
- **Team Size**: 1 (Solo developer)
- **Recommendations**: Personsas No, Units No, NFR Yes
- **Build**: `go build ./...` ผ่าน, `go test ./...` ทุก package ผ่าน, coverage 78.7%
- **Current Mode**: parallel implementation, 16/20 tasks complete (80%)

---

## Decision Questions

### D5-1: CI/CD Platform
**Question**: Which CI/CD platform will run your pipeline?

- 1) GitHub Actions — **Recommended** — เหมาะกับโปรเจกต์ Go ที่มี repository บน GitHub; ใช้ `make test` / `make build` ได้ตรงตัว; มี ecosystem สำหรับ Go ครบครัน
- 2) GitLab CI — ใช้ได้กับ GitLab repository
- 3) Azure DevOps — ใช้กับ Azure ecosystem
- 4) Other (please specify):

**Answer**: 1) GitHub Actions

---

### D5-2: Deployment Target
**Question**: Where does the application run?

- 1) Containers (Docker) — **Recommended** — โปรเจกต์นี้มี docker-compose.yml อยู่แล้ว; ใช้ bufconn สำหรับ gRPC integration test; เหมาะกับ deployment แบบ container
- 2) Serverless (AWS Lambda / Cloud Functions) — ไม่เหมาะกับ gRPC และโครงสร้าง hexagonal
- 3) Traditional VM / Bare metal — เยอะเกินความจำเป็นสำหรับ solo developer
- 4) Other (please specify):

**Answer**: 1) Containers (Docker)

---

### D5-3: Deployment Strategy
**Question**: How should new versions roll out?

- 1) Recreate — หยุด service เก่าก่อนเริ่ม service ใหม่ — ง่ายที่สุด แต่มี downtime เบา ๆ
- 2) Rolling Update — อัพเดททีละพอดเดอร์โดยอัตโนมัติ — **Recommended** — เหมาะกับ Docker Swarm / K8s; มี downtime น้อยกว่า; ทำได้กับ docker-compose ด้วยการควบคุม manual
- 3) Blue-Green — มีทั้งสอง version พร้อมกันแลกเปลี่ยน — **Recommended สำหรับความมั่นคง** — เหมาะกับ solo developer; ทันทีที่ swap เรียกใช้ version ใหม่; ถ้ามีปัญหา swap กลับได้ทันที
- 4) Canary — เผยแพร่ให้กลุ่มผู้ใช้เล็กก่อน — สำหรับ high-traffic แอปพลิเคชัน

**Answer**: 3) Blue-Green

---

### D5-4: Environments
**Question**: Which environments do you need?

- 1) dev + production — **Recommended** — Solo developer; dev สำหรับพัฒนาทุกวัน; production สำหรับใช้งานจริง
- 2) dev + staging + production — ใช้เมื่อมีทีมงานหลายคนหรือมีการทดสอบ UAT อย่างเป็นทางการ
- 3) dev only — ไม่มี environment การผลิต
- 4) Other (please specify):

**Answer**: 1) dev + production

---

### D5-5: Environment Promotion
**Question**: How are deployments promoted between environments?

- 1) Auto-deploy to dev on push; manual approval for staging/production — **Recommended** — อัตโนมัติเมื่อ push เข้า main → dev; ให้ผู้ดูแล hand-check ก่อน promotions ไป production
- 2) Manual approval สำหรับทุกการ deploy — ช้าแต่แน่นอน
- 3) CI pipeline อัตโนมัติทุกเฟส — เร็วแต่ต้องมั่นใจคุณภาพสูง
- 4) Other (please specify):

**Answer**: 1) Auto-deploy to dev on push; manual approval for production

---

### D5-6: Secrets Management
**Question**: How are secrets and credentials handled?

- 1) GitHub Secrets (หรือ GitLab Variables) — **Recommended** — โปรเจกต์นี้จะใช้ GitHub Actions; เก็บ JWT_SECRET, MONGO_URI, bcrypt salt/key ที่นี่; CI สามารถเข้าถึงได้เฉพาะ
- 2) .env files ใน repository — ไม่ปลอดภัยสำหรับ production; ใช้ได้แค่ development
- 3) Cloud provider secret management (AWS Secrets Manager, etc.) — เยอะเกินความจำเป็นสำหรับการ Deploy แบบ local
- 4) Other (please specify):

**Answer**: 1) GitHub Secrets

---

### D5-7: Rollback Strategy
**Question**: How do you recover from a bad deployment?

- 1) Blue-Green instant switch — **Recommended** — เนื่องจากเลือกใช้ Blue-Green deployment; swap กลับไปยัง version ก่อนทันทีที่พบปัญหา; ไม่มี downtime
- 2) Rolling back to previous version — ใช้สำหรับ Rolling Update; ต้องเก็บ image เวอร์ชันก่อนไว้
- 3) Manual rollback — คำสั่ง docker-compose down + up เวอร์ชันเก่า — สำหรับ solo developer ขนาดเล็ก
- 4) Other (please specify):

**Answer**: 1) Blue-Green instant switch

---

### D5-8: Monitoring & Alerts
**Question**: What signals indicate deployment health?

- 1) HTTP 5xx rate + response time p95 + health endpoint check — **Recommended** — ตรวจสอบ `/health` endpoint ที่มีอยู่แล้ว; ตั้งค่าการเตือนเมื่อ 5xx เพิ่มขึ้น; ติดตาม response time p95
- 2) ตัวชี้วัดด้านฐานข้อมูลเช่น query latency, connection count — มีประโยชน์แต่เป็น secondary
- 3) Log volume และ error rate เฉพาะ — ใช้ควบคุมร่วมกับข้อ 1
- 4) Other (please specify):

**Answer**: 1) HTTP 5xx rate + response time p95 + health endpoint check

---

### D5-9: Branch Strategy
**Question**: Which branches trigger which environments?

- 1) `main` → production, `develop` → staging, feature branches → dev/preview — เหมาะกับทีมงานหลายคน
- 2) `main` → production, feature branches → dev — **Recommended** — Solo developer; push เข้า main → deploy อัตโนมัติไป production (หรือ经过 manual approval ก่อน); feature branch → dev ทันที
- 3) ทุก branch → dev — เร็วที่สุด แต่ไม่มีการควบคุมสู่ production
- 4) Other (please specify):

**Answer**: 2) `main` → production, feature branches → dev

---

### D5-10: Infrastructure as Code
**Question**: Should infrastructure be provisioned alongside the app?

- 1) Extend existing docker-compose.yml — **Recommended** — โปรเจกต์มี docker-compose.yml อยู่แล้ว (api + mongo:7 + healthcheck); สามารถเพิ่ม service อื่น ๆ เข้าไปได้; ไม่จำเป็นต้องใช้ Terraform/CDK สำหรับ solo developer
- 2) สร้าง Terraform/CDK สำหรับคลาวด์ — โอกาสต้องการมีมากกว่าแค่ local development
- 3) ข้ามไป — ใช้ docker-compose คงไว้สำหรับทุกเฟส
- 4) Other (please specify):

**Answer**: 1) Extend existing docker-compose.yml

---

## Decisions Summary
<!-- Machine-readable compact summary. Downstream phases: read ONLY this section. -->

- D5-1 CI Platform: GitHub Actions
- D5-2 Deployment Target: Containers (Docker)
- D5-3 Deployment Strategy: Blue-Green
- D5-4 Environments: dev + production
- D5-5 Environment Promotion: Auto-deploy to dev on push; manual approval for production
- D5-6 Secrets Management: GitHub Secrets
- D5-7 Rollback Strategy: Blue-Green instant switch
- D5-8 Monitoring & Alerts: HTTP 5xx rate + response time p95 + health endpoint check
- D5-9 Branch Strategy: `main` → production, feature branches → dev
- D5-10 IaC: Extend existing docker-compose.yml

---

**Next step**: Proceed to generate pipeline configuration (`actions/generate.md`)