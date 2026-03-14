# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

JoyTime is a family task/reward management app with token economy. Parents assign tasks, children complete them for tokens, tokens are exchanged for rewards. Multi-platform (Telegram, web, mobile).

## Commands (via mise)

```bash
mise run build              # Build to ./build/
mise run run                # Build and run (port 8080)
mise run test               # Run all tests (starts DB via docker-compose)
mise run test-coverage      # Coverage report
mise run fmt                # Format code
mise run lint               # golangci-lint + go vet
mise run ci                 # Format + lint + test

mise run db:up              # Start PostgreSQL container
mise run db:down            # Stop PostgreSQL
mise run db:reset           # Full reset with test data
mise run db:fill            # Build app, start DB, seed test data
mise run db:shell           # psql into the database
```

Running a single test: `go test ./internal/handlers/ -run TestName -v` (requires DB running via `mise run db:up`).

## Architecture

Three-layer architecture under `internal/`:

- **handlers/** — HTTP layer. `APIHandler` struct owns all endpoints. Routes registered in `http.go`, grouped by resource in `handler_*.go` files. Uses stdlib `net/http` (no framework). Authentication via `X-User-ID` header in middleware.
- **domain/** — Business logic. Service structs (`TaskService`, `RewardService`, `TokenService`, `UserService`, `FamilyService`) orchestrated through `Services`. Every service method receives `AuthContext` (userID, role, familyUID) for authorization. Casbin enforces RBAC+ABAC (role permissions scoped to family).
- **database/** — GORM connection setup. **models/** — GORM schema definitions.

Request flow: HTTP → validation → auth middleware (builds AuthContext) → handler → domain service (Casbin check + business rules) → GORM → PostgreSQL.

## Key Conventions

- No global variables — all dependencies injected
- No shell scripts — use mise for task running and dependency management exclusively
- No backward compatibility constraints — refactor freely
- Always run tests after changes
- Use Go constants for error/log messages (see `constants.go`)
- Casbin RBAC model is defined programmatically in `domain/casbin_auth.go` (in-memory, no policy files)
- Tests use real PostgreSQL (docker-compose), not mocks
- Token operations maintain audit trail via `TokenHistory`

## Environment

Configured via `.env` file (see `.env` or `mise run setup-env`). Key vars: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `TOKEN` (Telegram bot), `DEBUG` (enables debug logging).
