# Deployment Decisions (D5)

## Context Summary
- **Feature**: user-management-api — REST API จัดการผู้ใช้ด้วย Go, MongoDB และ JWT authentication
- **Stack**: Go 1.26.4 / Fiber v2 / MongoDB (mongo-driver v2) / JWT (HMAC HS256) / gRPC
- **Architecture**: Hexagonal (Ports & Adapters) — stateless API + shared MongoDB
- **Infrastructure**: Docker + docker-compose (api + mongo:7 + healthcheck); build ผ่าน `make build` → binary `bin/api_user_mgr`
- **CI ที่มีอยู่**: `.github/workflows/deploy.yml` — GitHub Actions (build/test/dev deploy, prod เป็น commented-out)

---

## Decision Questions

### D5-1: CI/CD Platform
**Question**: CI/CD platform ใดที่ใช้รัน pipeline ของโปรเจกต์?
- 1) GitHub Actions — ตรงกับ CI ที่มีอยู่แล้ว (`deploy.yml`) **(Recommended)**
- 2) GitLab CI
- 3) Jenkins / อื่นๆ
- 4) Other (please specify): _______

**Answer**: 

---

### D5-2: Deployment Target
**Question**: แอปพลิเคชันนี้ deploy ไปที่ไหน?
- 1) Containers (Docker / docker-compose) — ตรงกับ stack ที่มีอยู่ (Dockerfile + docker-compose.yml) **(Recommended)**
- 2) Kubernetes (EKS/GKE/AKS)
- 3) Serverless (Lambda/Cloud Functions)
- 4) Other (please specify): _______

**Answer**: 

---

### D5-3: Deployment Strategy
**Question**: เวอร์ชันใหม่ควร rollout แบบไหน?
- 1) Blue-Green — เปิด instance ใหม่คู่กันแล้วสลับ traffic; rollback กลับได้ทันที — เหมาะกับ API ที่ต้องการ uptime **(Recommended)**
- 2) Rolling update
- 3) Recreate (stop แล้ว start ใหม่)
- 4) Other (please specify): _______

**Answer**: 

---

### D5-4: Environments
**Question**: ต้องมี environments กี่แบบ?
- 1) dev + production — solo team, 2 environments ก็เพียงพอ **(Recommended)**
- 2) dev + staging + production
- 3) dev เท่านั้น
- 4) Other (please specify): _______

**Answer**: 

---

### D5-5: Environment Promotion
**Question**: การ promote ระหว่าง environments เป็นแบบไหน?
- 1) Auto-deploy ไป dev เมื่อ push; manual approval สำหรับ production **(Recommended)**
- 2) Manual ทั้งหมด
- 3) Auto ทั้งหมด
- 4) Other (please specify): _______

**Answer**: 

---

### D5-6: Secrets Management
**Question**: จัดการ secrets/credentials อย่างไร?
- 1) GitHub Secrets (`${{ secrets.NAME }}`) — platform-native กับ GitHub Actions **(Recommended)**
- 2) AWS Secrets Manager / cloud-native
- 3) Vault
- 4) Other (please specify): _______

**Answer**: 

---

### D5-7: Rollback Strategy
**Question**: กู้คืนจาก deployment ที่ล้มเหลวด้วยวิธีไหน?
- 1) Blue-Green instant switch — สลับกลับไปเวอร์ชันก่อนหน้าได้ทันที **(Recommended)**
- 2) Redeploy เวอร์ชันก่อนหน้า
- 3) Halt rollout
- 4) Other (please specify): _______

**Answer**: 

---

### D5-8: Monitoring & Alerts
**Question**: สัญญาณอะไรที่บ่งชี้ว่า deployment ปกติ/ล้มเหลว?
- 1) HTTP 5xx rate + response time p95 + health endpoint check — เริ่มจาก health check (`GET /health`) ที่มีอยู่ **(Recommended)**
- 2) เพิ่ม error log spike monitoring
- 3) Full observability stack
- 4) Other (please specify): _______

**Answer**: 

---

### D5-9: Branch Strategy
**Question**: branch ไหน trigger deploy ไป environment ไหน?
- 1) `main` → production, feature branches → dev **(Recommended)**
- 2) `main` → dev+prod ด้วยกัน
- 3) `main` → dev, release branch → prod
- 4) Other (please specify): _______

**Answer**: 

---

### D5-10: Infrastructure as Code
**Question**: ควร provision infrastructure ควบคู่ไปกับแอปไหม?
- 1) Extend `docker-compose.yml` ที่มีอยู่ — compose ครอบคลุม api + mongo + healthcheck อยู่แล้ว **(Recommended)**
- 2) Terraform/CDK (สำหรับ cloud)
- 3) ไม่ต้องใช้ IaC
- 4) Other (please specify): _______

**Answer**: 

---

## Decisions Summary
<!-- Machine-readable compact summary. Downstream phases: read ONLY this section. -->
<!-- Auto-populated after user fills answers. One line per decision. -->
- D5-1 CI/CD Platform: GitHub Actions
- D5-2 Deployment Target: Containers (Docker / docker-compose)
- D5-3 Deployment Strategy: Blue-Green
- D5-4 Environments: dev + production
- D5-5 Promotion: auto-dev, manual approval for production
- D5-6 Secrets: GitHub Secrets
- D5-7 Rollback: Blue-Green instant switch
- D5-8 Monitoring: HTTP 5xx rate + response time p95 + health endpoint check
- D5-9 Branch Strategy: main → production, feature branches → dev
- D5-10 IaC: extend docker-compose.yml