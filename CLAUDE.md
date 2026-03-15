# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

JoyTime is a family task/reward management app with token economy. Parents assign tasks, children complete them for tokens, tokens are exchanged for rewards. Parents can also apply penalties. Primary UI is a Telegram bot; HTTP REST API available for web/mobile integration.

## Commands (via mise)

```bash
mise run build              # Build to ./build/
mise run run                # Build and run (port 8080 + Telegram bot)
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
mise run db:dump [file]     # Dump database to file (default: dump.sql)
mise run db:restore [file]  # Restore database from file (default: dump.sql)
```

Running a single test: `go test ./internal/handlers/ -run TestName -v` (requires DB running via `mise run db:up`).

## Architecture

Three-layer architecture under `internal/`:

- **handlers/** — HTTP layer. `APIHandler` struct owns all endpoints. Routes registered in `http.go`, grouped by resource in `handler_*.go` files. Uses stdlib `net/http` (no framework). Authentication via `X-User-ID` header in middleware.
- **domain/** — Business logic. Service structs (`TaskService`, `RewardService`, `PenaltyService`, `TokenService`, `UserService`, `FamilyService`, `InviteService`) orchestrated through `Services`. Every service method receives `AuthContext` (userID, role, familyUID) for authorization. Casbin enforces RBAC+ABAC (role permissions scoped to family).
- **telegram/** — Telegram bot layer. Uses `gopkg.in/telebot.v4`. `Bot` struct wraps telebot with domain `Services`. State management via `InputState`/`InputContext` fields on `Users` model. Number selection uses inline keyboard grids (7 columns).
- **database/** — GORM connection setup (SQLite default, PostgreSQL optional). **models/** — GORM schema definitions.

Request flow (HTTP): HTTP → validation → auth middleware (builds AuthContext) → handler → domain service (Casbin check + business rules) → GORM → DB.

Request flow (Telegram): Update → handleCallback/handleText → domain service (Casbin check + business rules) → GORM → DB → response with inline keyboard.

Registration flow: `/start` → "Create family" (creates parent + family) or "Enter invite code" (deep link `t.me/bot?start=CODE` or manual entry). Invite codes are one-time, carry role (parent/child) and familyUID. Family codes replaced by invite links.

## Key Conventions

- No global variables — all dependencies injected
- No shell scripts — use mise for task running and dependency management exclusively
- No backward compatibility constraints — refactor freely
- Always run tests after changes
- Use Go constants for error/log messages (see `constants.go`)
- Casbin RBAC model is defined programmatically in `domain/casbin_auth.go` (in-memory, no policy files)
- Tests use real PostgreSQL (docker-compose), not mocks
- Token operations maintain audit trail via `TokenHistory`
- Tasks are repeatable — reset to `new` after parent approval
- Entities (tasks, rewards, penalties) sorted by tokens descending
- User deletion cascades to Tokens and TokenHistory (clean re-add)
- Soft-deleted users are restored (not duplicated) on re-registration
- Partial unique indexes (WHERE deleted_at IS NULL) instead of GORM uniqueIndex tags, managed manually in `database/client.go` post-migration

## Environment

Configured via `.env` file (see `.env` or `mise run setup-env`).

| Variable | Default | Description |
|----------|---------|-------------|
| `TOKEN` | (required) | Telegram bot token |
| `DB_TYPE` | `sqlite` | Database type: `sqlite` or `postgres` |
| `DB_PATH` | `joytime.db` | SQLite file path |
| `PGHOST` | — | PostgreSQL host (required if DB_TYPE=postgres) |
| `PGPORT` | — | PostgreSQL port |
| `PGUSER` | — | PostgreSQL user |
| `PGPASSWORD` | — | PostgreSQL password |
| `PGDATABASE` | — | PostgreSQL database name |
| `DEBUG` | — | Enables debug logging if set |
