# Casbin RBAC+ABAC Implementation

## Overview

JoyTime uses [Casbin](https://casbin.org/) for authorization with a programmatic RBAC+ABAC model. Policies are defined in code (source of truth), stored in memory (no database persistence for policies), and enforce both role-based permissions and family-scoped access control.

## Architecture

```
┌──────────────┐    ┌──────────────┐    ┌──────────────────┐
│  HTTP Layer  │───▶│ Domain Layer │───▶│  CasbinAuth      │
│  handlers/   │    │ domain/      │    │  Service          │
│              │    │              │    │                    │
│ • Routing    │    │ • Business   │    │ • RequirePermission│
│ • Validation │    │   Logic      │    │ • CheckPermission  │
│ • Auth MW    │    │ • Services   │    │ • In-memory model  │
└──────────────┘    └──────────────┘    └──────────────────┘
```

## Model Definition

The Casbin model is defined programmatically in `domain/casbin_auth.go` (no config files):

```
[request_definition]
r = sub, obj, act, familyCtx, resourceFamily

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act && r.familyCtx == r.resourceFamily
```

Key: the matcher includes `r.familyCtx == r.resourceFamily` — this is the ABAC part that enforces family isolation. A parent in family A cannot access resources in family B, even though they have parent-level permissions.

## Permissions Matrix

| Resource | Action        | Parent | Child | Notes                                   |
|----------|---------------|--------|-------|-----------------------------------------|
| tasks    | create        | +      | -     | Only parents can create tasks           |
| tasks    | read          | +      | +     | All can view family tasks               |
| tasks    | update        | +      | -     | Only parents can modify tasks           |
| tasks    | delete        | +      | -     | Only parents can delete tasks           |
| tasks    | complete      | +      | +     | Children: "check", Parents: "completed" |
| rewards  | create        | +      | -     | Only parents can create rewards         |
| rewards  | read          | +      | +     | All can view available rewards          |
| rewards  | update        | +      | -     | Only parents can modify rewards         |
| rewards  | delete        | +      | -     | Only parents can delete rewards         |
| rewards  | claim         | -      | +     | Only children can claim rewards         |
| penalties| create        | +      | -     | Only parents can create penalties       |
| penalties| read          | +      | +     | All can view family penalties           |
| penalties| update        | +      | -     | Only parents can modify penalties       |
| penalties| delete        | +      | -     | Only parents can delete penalties       |
| penalties| apply         | +      | -     | Only parents can apply penalties        |
| invites  | create        | +      | -     | Only parents can create invite codes    |
| tokens   | read          | +      | +     | Own tokens only for children            |
| tokens   | read_others   | +      | -     | Parents can see all family tokens       |
| tokens   | add           | +      | -     | Only parents can give tokens            |
| tokens   | manual_adjust | +      | -     | Manual token adjustments                |
| users    | read          | +      | +     | Own profile for children                |
| users    | read_others   | +      | -     | Parents can view all family members     |
| users    | update        | +      | +     | Own profile for children                |
| users    | update_others | +      | -     | Parents can update children profiles    |
| users    | delete        | +      | -     | Parents can delete family members       |
| users    | create        | +      | -     | Parents can add family members          |
| family   | read          | +      | +     | All can view family info                |
| family   | update        | +      | -     | Only parents can update family          |
| family   | create        | +      | -     | Family creation during registration     |

## Usage in Services

### Permission check
```go
func (s *TaskService) CreateTask(ctx context.Context, authCtx *AuthContext, task *models.Tasks) error {
    // Checks role permission AND family match in one call
    if err := s.auth.RequirePermission(authCtx, "tasks", "create", task.FamilyUID); err != nil {
        return err // Returns ErrUnauthorized with descriptive message
    }
    return s.db.WithContext(ctx).Create(task).Error
}
```

### Self vs others distinction
For resources where children can access their own data but not others':
```go
if authCtx.UserID == targetUserID {
    err = s.auth.RequirePermission(authCtx, "tokens", "read", familyUID)
} else {
    err = s.auth.RequirePermission(authCtx, "tokens", "read_others", familyUID)
}
```

## Policy Initialization

Policies are defined as code in `InitializePolicies()` and loaded into the in-memory Casbin enforcer on startup. There are no policy files or database tables for Casbin rules.

```go
policyList := [][]string{
    {"parent", "tasks", "create"},
    {"parent", "tasks", "read"},
    // ... all policies defined here
    {"child", "tasks", "read"},
    {"child", "tasks", "complete"},
    // ...
}
```

To add a new permission: add an entry to `policyList` in `domain/casbin_auth.go` and restart.

## Testing

Tests use real Casbin enforcement against a real PostgreSQL database:

```go
// Test that child cannot create tasks
childCtx := &domain.AuthContext{
    UserID:    child.UserID,
    UserRole:  domain.RoleChild,
    FamilyUID: family.UID,
}
err := services.TaskService.CreateTask(ctx, childCtx, task)
assert.Contains(t, err.Error(), "unauthorized")
```

Test suites:
- `TestCasbinParentPermissions` — parents can perform parent-only actions
- `TestCasbinChildPermissions` — children are allowed/blocked appropriately
- `TestCasbinFamilyIsolation` — cross-family access is denied
- `TestRewardRBAC` — reward-specific permission enforcement
- `TestRewardFamilyIsolation` — reward cross-family isolation
