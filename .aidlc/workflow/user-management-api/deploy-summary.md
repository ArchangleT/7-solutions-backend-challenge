# Deployment Summary — user-management-api

**Date**: 2026-09-13
**CI/CD Platform**: GitHub Actions
**Deployment Target**: Containers (Docker / docker-compose)
**Strategy**: Blue-Green

## Pipeline

| Stage | Trigger | Actions |
|---|---|---|
| Build | push + PR | `go build ./...` + docker image build/push → GHCR (`ghcr.io/<owner>/user-management-api:<sha>`) |
| Test | After build | `go test ./...` + coverage gate ≥ 80% non-generated |
| Quality | After test | `go vet ./...` + gofmt check |
| Deploy (dev) | feature branch push | auto — `docker compose up -d api` + health smoke (`GET /health`) |
| Deploy (production) | `main` push | manual approval (protected environment) — blue-green via `scripts/deploy.sh` |

## Environments

| Environment | Branch | Promotion | URL |
|---|---|---|---|
| dev | feature branches | auto | http://<dev-host>:8080 (TBD per host) |
| production | `main` | manual approval (GitHub required reviewers) | http://<prod-host>:8080 (TBD per host) |

## Files Generated

| File | Purpose |
|---|---|
| `.github/workflows/deploy.yml` | CI/CD pipeline (build/test/quality/dev/prod blue-green) |
| `docker-compose.yml` | extend — `IMAGE_NAME:IMAGE_TAG` + `api-green` standby (profile `deploy`) + `APP_ENV` |
| `scripts/deploy.sh` | blue-green deploy + rollback helper |
| `.env.dev.example` / `.env.production.example` | environment templates (secrets เป็น placeholder) |
| `.dockerignore` | build context optimization |
| `.aidlc/workflow/user-management-api/decisions-deploy.md` | D5 decision gate |
| `.aidlc/workflow/user-management-api/deploy-summary.md` | this summary |

## Secrets Required

| Secret | Environment | Where to Configure |
|---|---|---|
| `JWT_SECRET` | production | GitHub repo → Settings → Secrets and variables → Actions |
| `MONGO_URI` | production (optional override) | GitHub repo secrets (default ใน compose ชี้ไป `mongo` service) |

> Dev ใช้ค่า default ใน `docker-compose.yml`; production ต้องตั้ง `JWT_SECRET` ก่อน deploy-prod รอบแรก

## Rollback

- **Strategy**: Blue-Green instant switch
- **Trigger**: `APP_ENV=prod IMAGE_NAME=<name> IMAGE_TAG=<previous-sha> JWT_SECRET=<...> bash scripts/deploy.sh rollback`
- **Recovery time**: seconds (เปลี่ยนเพียง container ต้นทาง traffic)

## Post-Deployment Checklist

- [x] Configure secrets in CI (รายการข้างบน — ต้องตั้งที่ GitHub เอง)
- [ ] Verify deployment target access/credentials (self-hosted runner / DOCKER_HOST)
- [ ] Set up monitoring/alerting: `GET /health` smoke มีใน pipeline แล้ว; 5xx rate + p95 ยัง pending (D5-8)
- [ ] Run first deployment to dev (push feature branch)
- [ ] Verify health checks respond (wget/curl smoke ในทุก step)
- [ ] Document runbook for on-call (optional)