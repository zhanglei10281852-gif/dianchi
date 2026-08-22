# Dianchi Battery Recovery Platform

Dianchi coordinates traceable intake, safety inspection, dismantling, material recovery and compliance certificates for retired lithium batteries. It is a backend operational system with authenticated operator and supervisor roles.

## Runtime

Set `Dianchi_DATABASE_URL` (defaults to `file:dianchi.db`) and `PORT` (defaults to `8080`). Run `GOTOOLCHAIN=local go run ./cmd/server`.

The API exposes `/healthz`, `/readyz`, `/v1/auth/login`, `/v1/lots`, `/v1/lots/{id}/inspection`, `/v1/lots/{id}/reserve`, `/v1/lots/{id}/recover`, `/v1/lots/{id}/certificate`, and audit queries. Every request carries a request id and context through HTTP, service, repository and worker layers.

## Workflow

An intake lot moves through `received -> inspected -> reserved -> dismantling -> recovered -> certified`. Safety findings can quarantine a lot and prevent downstream operations. Recovery writes material balances, an immutable audit event and an idempotency record in one transaction. A worker retries certificate publication with bounded backoff and records permanent failures.

## Verification

`go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, and `go run .agents/skills/go-base-project-create/scripts/measure_project.go -root . -enforce` are the supported checks.
