package domain

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

// InMemoryAdapter is a simple in-memory adapter for Casbin
type InMemoryAdapter struct{}

// Ensure InMemoryAdapter implements persist.Adapter
// Compile-time Safety:
// Without this line, if you forgot to implement a required method
// (like LoadPolicy, SavePolicy, etc.), you'd only discover the error
// when you try to use InMemoryAdapter as a persist.Adapter at runtime.
// With this line, the code won't even compile if you're missing methods.
var _ persist.Adapter = &InMemoryAdapter{}

// LoadPolicy loads all policy rules from memory (no-op since we manage in code)
func (a *InMemoryAdapter) LoadPolicy(model model.Model) error {
	return nil
}

// SavePolicy saves all policy rules to memory (no-op since we don't persist)
func (a *InMemoryAdapter) SavePolicy(model model.Model) error {
	return nil
}

// AddPolicy adds a policy rule to memory (no-op since we manage in code)
func (a *InMemoryAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	return nil
}

// RemovePolicy removes a policy rule from memory (no-op since we manage in code)
func (a *InMemoryAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return nil
}

// RemoveFilteredPolicy removes policy rules that match the filter from memory
func (a *InMemoryAdapter) RemoveFilteredPolicy(
	sec string,
	ptype string,
	fieldIndex int,
	fieldValues ...string,
) error {
	return nil
}

// CasbinAuthService handles authorization using Casbin
type CasbinAuthService struct {
	enforcer *casbin.Enforcer
	logger   *log.Logger
	db       *gorm.DB
}

// NewCasbinAuthService creates a new Casbin-based authorization service
func NewCasbinAuthService(db *gorm.DB, logger *log.Logger) (*CasbinAuthService, error) {
	// Create in-memory adapter - no persistence needed since code is source of truth
	adapter := &InMemoryAdapter{}

	// Create the model programmatically instead of loading from file
	// This defines the Casbin ABAC model configuration with family context:
	// - request_definition: format for authorization requests (subject, object, action, familyCtx, resourceFamily)
	// - policy_definition: format for permission rules (subject, object, action)
	// - role_definition: format for role inheritance (user, role)
	// - policy_effect: allow access if any matching rule allows it
	// - matchers: logic to match requests against policies using role inheritance and family ownership
	//   g(r.sub, p.sub) checks if request subject has the policy role
	//   r.obj == p.obj && r.act == p.act checks object and action match
	//   r.familyCtx == r.resourceFamily ensures user can only access their family's resources
	modelText := `
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
`

	// Create model from string
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin model: %w", err)
	}

	// Create the enforcer with programmatic model and in-memory adapter
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	service := &CasbinAuthService{
		enforcer: enforcer,
		logger:   logger,
		db:       db,
	}

	// Always initialize policies from code (source of truth)
	err = service.InitializePolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policies: %w", err)
	}

	return service, nil
}

// InitializePolicies initializes the basic policies from code (source of truth)
func (cas *CasbinAuthService) InitializePolicies() error {
	cas.logger.Info("Initializing Casbin policies from code")

	// Clear any existing policies
	cas.enforcer.ClearPolicy()

	// Define policies in code - this is the source of truth
	policyList := [][]string{
		// Parent permissions
		{"parent", "tasks", "create"},
		{"parent", "tasks", "read"},
		{"parent", "tasks", "update"},
		{"parent", "tasks", "delete"},
		{"parent", "tasks", "complete"},
		{"parent", "rewards", "create"},
		{"parent", "rewards", "read"},
		{"parent", "rewards", "update"},
		{"parent", "rewards", "delete"},
		{"parent", "tokens", "read"},
		{"parent", "tokens", "read_others"},
		{"parent", "tokens", "add"},
		{"parent", "tokens", "manual_adjust"},
		{"parent", "users", "read"},
		{"parent", "users", "read_others"},
		{"parent", "users", "update"},
		{"parent", "users", "update_others"},
		{"parent", "users", "delete"},
		{"parent", "users", "create"},
		{"parent", "family", "read"},
		{"parent", "family", "update"},
		{"parent", "family", "create"},

		// Child permissions
		{"child", "tasks", "read"},
		{"child", "tasks", "complete"},
		{"child", "rewards", "read"},
		{"child", "rewards", "claim"},
		{"child", "tokens", "read"},
		{"child", "users", "read"},
		{"child", "users", "update"},
		{"child", "family", "read"},
	}

	// Add policies
	for _, policy := range policyList {
		_, err := cas.enforcer.AddPolicy(policy[0], policy[1], policy[2])
		if err != nil {
			return fmt.Errorf("failed to add policy %v: %w", policy, err)
		}
	}

	cas.logger.Info("Casbin policies initialized successfully from code")
	return nil
}

// CheckPermission checks if a user has permission and family access to a resource
func (cas *CasbinAuthService) CheckPermission(
	authCtx *AuthContext,
	resource, action, resourceFamilyUID string,
) (bool, error) {
	// Check permission using Casbin with family context
	allowed, err := cas.enforcer.Enforce(
		string(authCtx.UserRole), // sub (role)
		resource,                    // obj
		action,                      // act
		authCtx.FamilyUID,        // familyCtx (user's family)
		resourceFamilyUID,           // resourceFamily (resource's family)
	)
	if err != nil {
		return false, fmt.Errorf("failed to enforce policy: %w", err)
	}

	cas.logger.Debug("Permission check",
		"user_id", authCtx.UserID,
		"role", authCtx.UserRole,
		"resource", resource,
		"action", action,
		"user_family", authCtx.FamilyUID,
		"resource_family", resourceFamilyUID,
		"allowed", allowed)

	return allowed, nil
}

// RequirePermission checks permission and family access, returns error if not allowed
func (cas *CasbinAuthService) RequirePermission(
	authCtx *AuthContext,
	resource, action, resourceFamilyUID string,
) error {
	allowed, err := cas.CheckPermission(authCtx, resource, action, resourceFamilyUID)
	if err != nil {
		return err
	}

	if !allowed {
		cas.logger.Warn("Authorization failed",
			"user_id", authCtx.UserID,
			"role", authCtx.UserRole,
			"resource", resource,
			"action", action,
			"user_family", authCtx.FamilyUID,
			"resource_family", resourceFamilyUID)

		return fmt.Errorf(
			"%w: %s cannot %s %s (family access denied)",
			ErrUnauthorized,
			authCtx.UserRole,
			action,
			resource,
		)
	}

	return nil
}
