package gonfig

import (
	"os"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exprTestConfig demonstrates vm.Program configuration fields
type exprTestConfig struct {
	// Basic vm.Program field
	FilterExpr *vm.Program `env:"FILTER_EXPR"`

	// Required expr field
	ValidationExpr *vm.Program `env:"VALIDATION_EXPR" required:"true"`

	// Expr with default value
	DefaultExpr *vm.Program `env:"DEFAULT_EXPR" default:"true"`

	// Slice of expressions
	RuleExprs []*vm.Program `env:"RULE_EXPRS"`

	// Nested struct with expr
	Rules struct {
		AccessExpr *vm.Program `env:"ACCESS_EXPR"`
		AdminExpr  *vm.Program `env:"ADMIN_EXPR" default:"user.role == 'admin'"`
	}
}

func TestExprBasicParsing(t *testing.T) {
	os.Setenv("FILTER_EXPR", "user.age >= 18 && user.verified")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("FILTER_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.NotNil(t, cfg.FilterExpr)

	// Test that the expression can be executed
	env := map[string]interface{}{
		"user": map[string]interface{}{
			"age":      25,
			"verified": true,
		},
	}

	result, err := expr.Run(cfg.FilterExpr, env)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test with failing condition
	env["user"].(map[string]interface{})["age"] = 16
	result, err = expr.Run(cfg.FilterExpr, env)
	require.NoError(t, err)
	assert.False(t, result.(bool))
}

func TestExprWithDefault(t *testing.T) {
	// Don't set DEFAULT_EXPR, should use default value
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer os.Unsetenv("VALIDATION_EXPR")

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.NotNil(t, cfg.DefaultExpr)

	// Test that the default expression works
	result, err := expr.Run(cfg.DefaultExpr, nil)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExprNestedStruct(t *testing.T) {
	os.Setenv("ACCESS_EXPR", "'read' in user.permissions")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("ACCESS_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.NotNil(t, cfg.Rules.AccessExpr)
	require.NotNil(t, cfg.Rules.AdminExpr) // Should have default value

	// Test access expression
	env := map[string]interface{}{
		"user": map[string]interface{}{
			"permissions": []string{"read", "write"},
		},
	}

	result, err := expr.Run(cfg.Rules.AccessExpr, env)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test admin expression (default)
	adminEnv := map[string]interface{}{
		"user": map[string]interface{}{
			"role": "admin",
		},
	}

	result, err = expr.Run(cfg.Rules.AdminExpr, adminEnv)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test admin expression with non-admin
	adminEnv["user"].(map[string]interface{})["role"] = "user"
	result, err = expr.Run(cfg.Rules.AdminExpr, adminEnv)
	require.NoError(t, err)
	assert.False(t, result.(bool))
}

func TestExprSlice(t *testing.T) {
	os.Setenv("RULE_EXPRS", "user.age >= 18,user.verified == true,user.role == 'admin' || user.role == 'moderator'")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("RULE_EXPRS")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.Len(t, cfg.RuleExprs, 3)

	// Test each expression in the slice
	env := map[string]interface{}{
		"user": map[string]interface{}{
			"age":      25,
			"verified": true,
			"role":     "admin",
		},
	}

	// Test age expression
	result, err := expr.Run(cfg.RuleExprs[0], env)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test verified expression
	result, err = expr.Run(cfg.RuleExprs[1], env)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test role expression
	result, err = expr.Run(cfg.RuleExprs[2], env)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExprInvalidSyntax(t *testing.T) {
	os.Setenv("FILTER_EXPR", "user.age ++ invalid syntax")
	defer os.Unsetenv("FILTER_EXPR")

	_, err := Load(exprTestConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile expression")
}

func TestExprRequired(t *testing.T) {
	// Don't set VALIDATION_EXPR which is required
	_, err := Load(exprTestConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required env \"VALIDATION_EXPR\" missing")
}

func TestExprComplexExpressions(t *testing.T) {
	os.Setenv("FILTER_EXPR", "all(users, {.age >= 18 && .verified}) && len(users) > 0")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("FILTER_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.NotNil(t, cfg.FilterExpr)

	// Test with valid data
	env := map[string]interface{}{
		"users": []map[string]interface{}{
			{"age": 25, "verified": true},
			{"age": 30, "verified": true},
		},
	}

	result, err := expr.Run(cfg.FilterExpr, env)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test with one unverified user
	env["users"].([]map[string]interface{})[1]["verified"] = false
	result, err = expr.Run(cfg.FilterExpr, env)
	require.NoError(t, err)
	assert.False(t, result.(bool))
}

func TestExprStringFunctions(t *testing.T) {
	// Use valid expr syntax with string operations
	os.Setenv("FILTER_EXPR", "user.email matches '^admin@.*' && user.name contains 'John'")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("FILTER_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.NotNil(t, cfg.FilterExpr)

	env := map[string]interface{}{
		"user": map[string]interface{}{
			"email": "admin@example.com",
			"name":  "John Doe",
		},
	}

	result, err := expr.Run(cfg.FilterExpr, env)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExprMathOperations(t *testing.T) {
	os.Setenv("FILTER_EXPR", "user.score > 85.5 && user.attempts <= 3")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("FILTER_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	require.NotNil(t, cfg.FilterExpr)

	env := map[string]interface{}{
		"user": map[string]interface{}{
			"score":    92.7,
			"attempts": 2,
		},
	}

	result, err := expr.Run(cfg.FilterExpr, env)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExprEmptyString(t *testing.T) {
	os.Setenv("FILTER_EXPR", "")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("FILTER_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	// Empty string should result in nil pointer
	assert.Nil(t, cfg.FilterExpr)
}

func TestExprSliceWithEmptyElements(t *testing.T) {
	os.Setenv("RULE_EXPRS", "user.age >= 18,,user.verified == true")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("RULE_EXPRS")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)
	// Should only parse 2 expressions, skipping the empty one
	require.Len(t, cfg.RuleExprs, 2)

	env := map[string]interface{}{
		"user": map[string]interface{}{
			"age":      25,
			"verified": true,
		},
	}

	// Test both expressions
	result, err := expr.Run(cfg.RuleExprs[0], env)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	result, err = expr.Run(cfg.RuleExprs[1], env)
	require.NoError(t, err)
	assert.True(t, result.(bool))
}

func TestExprInPrettyString(t *testing.T) {
	os.Setenv("FILTER_EXPR", "user.age >= 18")
	os.Setenv("VALIDATION_EXPR", "true") // Required field
	defer func() {
		os.Unsetenv("FILTER_EXPR")
		os.Unsetenv("VALIDATION_EXPR")
	}()

	cfg, err := Load(exprTestConfig{})
	require.NoError(t, err)

	prettyStr := PrettyString(cfg)
	assert.Contains(t, prettyStr, "FILTER_EXPR")
	// The expr.Program should be represented in some way
	assert.NotEmpty(t, prettyStr)
}

// Test expr.Program in a more realistic configuration scenario
type realisticExprConfig struct {
	Port int `env:"PORT" default:"8080"`

	// Access control expressions
	Access struct {
		AdminCheck     *vm.Program `env:"ADMIN_CHECK" default:"user.role == 'admin'"`
		RateLimitCheck *vm.Program `env:"RATE_LIMIT_CHECK" default:"user.requests_per_hour < 1000"`
		FeatureFlags   *vm.Program `env:"FEATURE_FLAGS" default:"user.beta_features == true"`
	}

	// Business rules
	Rules []*vm.Program `env:"BUSINESS_RULES"`
}

func TestRealisticExprUsage(t *testing.T) {
	os.Setenv("ADMIN_CHECK", "user.role in ['admin', 'super_admin'] && user.active == true")
	os.Setenv("BUSINESS_RULES", "order.total > 100,customer.tier == 'premium',product.in_stock == true")
	defer func() {
		os.Unsetenv("ADMIN_CHECK")
		os.Unsetenv("BUSINESS_RULES")
	}()

	cfg, err := Load(realisticExprConfig{})
	require.NoError(t, err)

	// Test admin check
	adminEnv := map[string]interface{}{
		"user": map[string]interface{}{
			"role":   "admin",
			"active": true,
		},
	}

	result, err := expr.Run(cfg.Access.AdminCheck, adminEnv)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test default rate limit check
	rateLimitEnv := map[string]interface{}{
		"user": map[string]interface{}{
			"requests_per_hour": 500,
		},
	}

	result, err = expr.Run(cfg.Access.RateLimitCheck, rateLimitEnv)
	require.NoError(t, err)
	assert.True(t, result.(bool))

	// Test business rules
	require.Len(t, cfg.Rules, 3)

	businessEnv := map[string]interface{}{
		"order":    map[string]interface{}{"total": 150.0},
		"customer": map[string]interface{}{"tier": "premium"},
		"product":  map[string]interface{}{"in_stock": true},
	}

	// Test each business rule
	for i, rule := range cfg.Rules {
		result, err := expr.Run(rule, businessEnv)
		require.NoError(t, err, "Failed to run business rule %d", i)
		assert.True(t, result.(bool), "Business rule %d should be true", i)
	}
}
