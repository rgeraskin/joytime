# Refactoring Summary

## What Changed

The codebase was refactored from a 2-layer architecture to a 3-layer architecture with Casbin RBAC+ABAC authorization.

### Before

```
internal/
├── api/           # HTTP handlers + business logic mixed together
│   ├── http.go
│   ├── families.go, users.go, tasks.go, rewards.go, tokens.go
│   └── ...
└── postgres/      # DB connection + models + seeding
    ├── client.go, schema.go, fill.go
```

- Global variables for DB and logger
- Business logic embedded in HTTP handlers
- No authorization layer
- Monolithic file structure

### After

```
internal/
├── handlers/      # HTTP layer only
│   ├── http.go, middleware.go
│   ├── handler_families.go, handler_users.go
│   ├── handler_tasks.go, tasks_business.go
│   ├── handler_rewards.go, handler_tokens.go
│   ├── types.go, constants.go, validation.go
│   └── *_test.go
├── domain/        # Business logic + authorization
│   ├── services.go
│   ├── service_task.go, service_reward.go
│   ├── service_token.go, service_user.go, service_family.go
│   ├── casbin_auth.go
│   ├── types.go, utils.go
├── database/      # GORM connection setup
│   ├── client.go, fill.go
└── models/        # GORM schema definitions
    └── schema.go
```

## Key Improvements

| Area | Before | After |
|------|--------|-------|
| Architecture | 2-layer (HTTP + DB) | 3-layer (HTTP + domain + DB) |
| Authorization | None | Casbin RBAC+ABAC (role + family scoping) |
| Business logic | Mixed into HTTP handlers | Isolated in domain services |
| Dependencies | Global variables | Dependency injection |
| Family isolation | Manual checks | Casbin enforcer with family context |
| Reward CRUD | Not implemented | Full CRUD with RBAC |
| Tests | Basic HTTP tests | 53 tests: service, RBAC, isolation, HTTP |

## Authorization

Added Casbin-based authorization with:
- **RBAC**: Role-based permissions (parent vs child)
- **ABAC**: Family-scoped access (users can only access their own family's data)
- **Programmatic model**: Defined in code (`domain/casbin_auth.go`), no config files
- **In-memory policies**: Loaded on startup, no database persistence for policies

## Test Coverage

| Test Suite | Count | What it covers |
|------------|-------|----------------|
| Parent permissions | 5 | Parents can create tasks/rewards, adjust tokens, etc. |
| Child permissions | 10 | Children can read/complete tasks, cannot create/delete |
| Family isolation | 6 | Cross-family access denied for tasks/users/tokens |
| Reward CRUD | 7 | Full lifecycle through service layer |
| Reward RBAC | 3 | Children cannot create/update/delete rewards |
| Reward isolation | 4 | Cross-family reward access denied |
| Reward claim | 3 | Token deduction, history, insufficient balance |
| Reward HTTP | 12 | All endpoints, auth, validation, error cases |
| Service integration | 4 | Family/user operations through service layer |
| UID generation | 3 | Family UID auto-generation and uniqueness |
| Handler names | 3 | Route registration and auth requirements |

All 53 tests pass against real PostgreSQL (no mocks).
