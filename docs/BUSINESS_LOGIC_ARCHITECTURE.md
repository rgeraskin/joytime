# Business Logic Architecture

## Overview

JoyTime uses a three-layer architecture where business rules and authorization are enforced in the domain layer via Casbin RBAC+ABAC.

## Architecture Layers

### 1. HTTP Layer (`internal/handlers/`)
- **Responsibility**: HTTP routing, request parsing, validation, response formatting
- **What it should NOT do**: Business logic, authorization decisions
- **Key files**: `http.go` (routes), `middleware.go` (auth), `handler_*.go` (endpoints), `validation.go` (input validation)

### 2. Domain/Business Logic Layer (`internal/domain/`)
- **Responsibility**: Business rules, role-based access control, family isolation
- **Key files**: `service_task.go`, `service_reward.go`, `service_token.go`, `service_user.go`, `service_family.go`, `casbin_auth.go`
- **What it does**:
  - Validates user permissions based on roles (parent/child) via Casbin
  - Enforces family boundaries (users can only access their family's data)
  - Implements business rules (e.g., "children cannot create tasks")
  - Coordinates complex operations (e.g., reward claim = permission check + balance check + token deduction + history record)

### 3. Data Layer (`internal/database/` + `internal/models/`)
- **Responsibility**: GORM connection setup, schema definitions, test data seeding
- **Key files**: `database/client.go` (connection + migration), `models/schema.go` (all GORM models)

## Request Flow

```
HTTP Request
  → handlers/middleware.go: AuthMiddleware extracts X-User-ID header
  → UserService.CreateAuthContext() builds AuthContext {UserID, UserRole, FamilyUID}
  → handler method parses request, validates input
  → domain service method receives AuthContext
  → CasbinAuthService.RequirePermission() checks RBAC+ABAC
  → business logic executes
  → GORM → PostgreSQL
  → handler formats response
```

## AuthContext

```go
type AuthContext struct {
    UserID    string
    UserRole  UserRole   // "parent" or "child"
    FamilyUID string
}
```

Every domain service method receives `AuthContext` as its first argument after `ctx`. This enables:
- **RBAC**: Casbin checks role-based permissions (parent can create tasks, child cannot)
- **ABAC**: Casbin checks family scoping (user's familyUID must match resource's familyUID)

## Business Rules

### Parent Permissions
- Create, read, update, delete tasks and rewards
- Manually adjust tokens for family members
- View all family member activities and token histories
- Approve child task completions (status → "completed")

### Child Permissions
- Read tasks and rewards in their family
- Complete tasks (status → "check" for parent approval)
- Read their own token balance and history
- Claim rewards (if they have enough tokens)
- Cannot create/delete tasks, cannot create/update/delete rewards
- Cannot give tokens to themselves or view others' tokens

### Family Isolation
- Users can only interact with data from their own family
- Casbin matcher enforces `user.FamilyUID == resource.FamilyUID`
- Cross-family access is denied at the Casbin level before any DB query

## Domain Services

### Services struct (`services.go`)
Aggregates all services and initializes Casbin:

```go
type Services struct {
    TaskService   *TaskService
    TokenService  *TokenService
    UserService   *UserService
    FamilyService *FamilyService
    RewardService *RewardService
    Auth          *CasbinAuthService
}
```

### Internal methods pattern
When a high-level authorized operation needs to perform sub-operations, it uses internal (unexported) methods that skip redundant permission checks. Example: `ClaimReward` (authorized via `rewards:claim`) calls `addTokens()` internally instead of `AddTokensToUser()` (which would re-check `tokens:add`).

## Authentication

Current implementation uses `X-User-ID` header. The middleware looks up the user in the database and builds `AuthContext` from their stored role and family. In production, replace with JWT/OAuth — only the middleware changes, the domain layer stays the same.
