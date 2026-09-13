# Deploy Summary — user-management-api

**Date**: 2026-09-13
**Decisions**: D5 — GitHub Actions · Containers (docker-compose) · Blue-Green · dev + production · GitHub Secrets
**Status**: ready for approval

## Pipeline: `.github/workflows/deploy.yml`

| Job | Trigger | Gate |
|---|---|---|
| `build` | push + PR | compile (`go build ./...`) + docker image build/push → GHCR |
| `test` | push + PR | `go test ./...` + coverage gate ≥ 80% non-generated |
| `quality` | push + PR | `go vet ./...` + gofmt check |
| `deploy-dev` | push (non-main branches) | auto — `docker compose up -d api` + health smoke (`GET /health`) |
| `deploy-prod` | push (main only) | **manual approval** — GitHub protected environment `production` → Blue-Green via `scripts/deploy.sh` |

**Environment promotion (D5-5)**: feature branches → dev (auto) · `main` → production (manual approval via GitHub environment protection rules)

## Deployment files

| File | Role |
|---|---|
| `docker-compose.yml` | extends existing — `image` tag substitution (`IMAGE_NAME:IMAGE_TAG`) + `api-green` standby (profile `deploy`) for blue-green; adds `APP_ENV` |
| `scripts/deploy.sh` | blue-green deploy: standby → pre-switch health + smoke → switch traffic → decommission; `rollback` = instant switch to previous `IMAGE_TAG` |
| `.env.dev.example` / `.env.production.example` | env templates — real secrets ตั้งใน GitHub Secrets (จี้ placeholder ไว้) |
| `.dockerignore` | build context optimization (excludes secrets, docs, test files, .aidlc) |

## Secrets (GitHub Secrets — ต้อง configure เอง)

- `JWT_SECRET` — production JWT signing secret (ยืนยันมาใช้ใน `deploy-prod`)
- `MONGO_URI` — ยังไม่ถูกอ้างใน workflow (ใช้ compose default `mongo`); รองรับ override ผ่าน env
- ตัวแปรอื่น (`IMAGE_NAME`) เป็น default จาก repo env — ปรับได้ตาม owner/repo

## Rollback strategy (D5-7)

Blue-green instant switch: ถ้า prod deploy ล้ม เหลือ `scripts/deploy.sh rollback` กับ `IMAGE_TAG=<previous-sha>` เปลี่ยนกลับทันที (seconds) — ขั้นตอนรันผ่าน job `Rollback procedure`

## Monitoring & health (D5-8)

- health endpoint check ในทุก deploy-step (`GET /health` → wget/curl)
- post-deploy smoke: standby health + active health หลัง switch
- กรณีต้องการสัญญาณเพิ่ม (5xx rate, p95): เพิ่ม monitoring stack — pending ตาม D5-8

## Assumptions

- Deploy jobs ใช้ runner ที่มี Docker access ไปยัง target host (self-hosted runner / `DOCKER_HOST`) — เหมาะกับ single-host docker-compose deployment
- `environment: production` ต้องตั้ง required reviewers ใน GitHub repo settings เพื่อบังคับ manual approval