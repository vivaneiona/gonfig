package gonfig

import (
	"os"
	"path/filepath"
	"testing"
)

// Test basic .env file loading
func TestLoadWithDotenv(t *testing.T) {
	type Config struct {
		Name     string `env:"DOTENV_APP_NAME" default:"default-app"`
		Port     int    `env:"DOTENV_PORT" default:"8080"`
		Debug    bool   `env:"DOTENV_DEBUG" default:"false"`
		APIKey   string `secret:"DOTENV_API_KEY" default:"default-key"`
		Database struct {
			Host string `env:"DOTENV_DB_HOST" default:"localhost"`
			Port int    `env:"DOTENV_DB_PORT" default:"5432"`
		}
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := `# This is a comment
DOTENV_APP_NAME=myapp
DOTENV_PORT=3000
DOTENV_DEBUG=true
DOTENV_API_KEY=secret123
DOTENV_DB_HOST=db.example.com
DOTENV_DB_PORT=5432
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	cfg, err := LoadWithDotenv(Config{})
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	// Verify values from .env file
	if cfg.Name != "myapp" {
		t.Errorf("Name = %q; want %q", cfg.Name, "myapp")
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d; want %d", cfg.Port, 3000)
	}
	if cfg.Debug != true {
		t.Errorf("Debug = %v; want %v", cfg.Debug, true)
	}
	if cfg.APIKey != "secret123" {
		t.Errorf("APIKey = %q; want %q", cfg.APIKey, "secret123")
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q; want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d; want %d", cfg.Database.Port, 5432)
	}
}

// Test .env file with custom path
func TestLoadWithDotenvCustomPath(t *testing.T) {
	type Config struct {
		Value string `env:"CUSTOM_VALUE" default:"default"`
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom .env file
	customEnvFile := filepath.Join(tempDir, "custom.env")
	envContent := "CUSTOM_VALUE=from_custom_env\n"
	if err := os.WriteFile(customEnvFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write custom .env file: %v", err)
	}

	cfg, err := LoadWithDotenv(Config{}, customEnvFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	if cfg.Value != "from_custom_env" {
		t.Errorf("Value = %q; want %q", cfg.Value, "from_custom_env")
	}
}

// Test precedence: environment variables > .env file > defaults
func TestDotenvPrecedence(t *testing.T) {
	type Config struct {
		EnvVar     string `env:"PRECEDENCE_ENV_VAR" default:"default_env"`
		DotenvVar  string `env:"PRECEDENCE_DOTENV_VAR" default:"default_dotenv"`
		DefaultVar string `env:"PRECEDENCE_DEFAULT_VAR" default:"default_only"`
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := `PRECEDENCE_ENV_VAR=from_dotenv_but_should_be_overridden
PRECEDENCE_DOTENV_VAR=from_dotenv
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Set an environment variable that should take precedence over .env
	t.Setenv("PRECEDENCE_ENV_VAR", "from_environment")

	cfg, err := LoadWithDotenv(Config{}, envFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	// Environment variable should win over .env
	if cfg.EnvVar != "from_environment" {
		t.Errorf("EnvVar = %q; want %q (env should override .env)", cfg.EnvVar, "from_environment")
	}

	// .env file should win over default
	if cfg.DotenvVar != "from_dotenv" {
		t.Errorf("DotenvVar = %q; want %q (.env should override default)", cfg.DotenvVar, "from_dotenv")
	}

	// Default should be used when neither env var nor .env is set
	if cfg.DefaultVar != "default_only" {
		t.Errorf("DefaultVar = %q; want %q (should use default)", cfg.DefaultVar, "default_only")
	}
}

// Test .env file with existing struct values
func TestDotenvWithExistingValues(t *testing.T) {
	type Config struct {
		ExistingValue string `env:"DOTENV_EXISTING" default:"default_existing"`
		ZeroValue     string `env:"DOTENV_ZERO" default:"default_zero"`
		DotenvOnly    string `env:"DOTENV_ONLY" default:"default_dotenv_only"`
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := `DOTENV_ZERO=from_dotenv_zero
DOTENV_ONLY=from_dotenv_only
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Pre-populate struct with some values
	cfg := &Config{
		ExistingValue: "pre_existing_value",
		ZeroValue:     "", // Zero value - should be overridden by .env
		DotenvOnly:    "", // Zero value - should be overridden by .env
	}

	_, err = LoadWithDotenv(cfg, envFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	// Existing non-zero value should be preserved (no .env value set)
	if cfg.ExistingValue != "pre_existing_value" {
		t.Errorf("ExistingValue = %q; want %q (existing value should be preserved)", cfg.ExistingValue, "pre_existing_value")
	}

	// Zero value should be overridden by .env
	if cfg.ZeroValue != "from_dotenv_zero" {
		t.Errorf("ZeroValue = %q; want %q (.env should override zero value)", cfg.ZeroValue, "from_dotenv_zero")
	}

	// Zero value should be overridden by .env
	if cfg.DotenvOnly != "from_dotenv_only" {
		t.Errorf("DotenvOnly = %q; want %q (.env should override zero value)", cfg.DotenvOnly, "from_dotenv_only")
	}
}

// Test .env file with nested structs
func TestDotenvWithNestedStructs(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"NESTED_DB_HOST" default:"localhost"`
		Port     int    `env:"NESTED_DB_PORT" default:"5432"`
		Password string `secret:"NESTED_DB_PASSWORD" default:"secret"`
	}

	type ServerConfig struct {
		Host string `env:"NESTED_SERVER_HOST" default:"0.0.0.0"`
		Port int    `env:"NESTED_SERVER_PORT" default:"8080"`
	}

	type AppConfig struct {
		Name     string `env:"NESTED_APP_NAME" default:"myapp"`
		Database DatabaseConfig
		Server   *ServerConfig // Test pointer struct too
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := `NESTED_APP_NAME=dotenv-app
NESTED_DB_HOST=dotenv-db-host
NESTED_DB_PORT=3306
NESTED_DB_PASSWORD=dotenv-secret
NESTED_SERVER_HOST=dotenv-server
NESTED_SERVER_PORT=9000
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	cfg, err := LoadWithDotenv(AppConfig{}, envFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	// Verify top-level field
	if cfg.Name != "dotenv-app" {
		t.Errorf("Name = %q; want %q", cfg.Name, "dotenv-app")
	}

	// Verify nested struct fields
	if cfg.Database.Host != "dotenv-db-host" {
		t.Errorf("Database.Host = %q; want %q", cfg.Database.Host, "dotenv-db-host")
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %d; want %d", cfg.Database.Port, 3306)
	}
	if cfg.Database.Password != "dotenv-secret" {
		t.Errorf("Database.Password = %q; want %q", cfg.Database.Password, "dotenv-secret")
	}

	// Verify pointer struct was allocated and populated
	if cfg.Server == nil {
		t.Fatal("Server should not be nil")
	}
	if cfg.Server.Host != "dotenv-server" {
		t.Errorf("Server.Host = %q; want %q", cfg.Server.Host, "dotenv-server")
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %d; want %d", cfg.Server.Port, 9000)
	}
}

// Test .env file with CSV slices
func TestDotenvWithSlices(t *testing.T) {
	type Config struct {
		Tags     []string  `env:"SLICE_TAGS" default:"default1,default2"`
		Ports    []int     `env:"SLICE_PORTS" default:"8080,8081"`
		Ratios   []float64 `env:"SLICE_RATIOS" default:"1.0,2.0"`
		Features []bool    `env:"SLICE_FEATURES" default:"true,false"`
		Secrets  []string  `secret:"SLICE_SECRETS" default:"secret1,secret2"`
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .env file
	envFile := filepath.Join(tempDir, ".env")
	envContent := `SLICE_TAGS=tag1,tag2,tag3
SLICE_PORTS=3000,4000,5000
SLICE_RATIOS=1.5,2.5,3.5
SLICE_FEATURES=false,true,false
SLICE_SECRETS=dotenv_secret1,dotenv_secret2,dotenv_secret3
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	cfg, err := LoadWithDotenv(Config{}, envFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	// Verify string slice
	expectedTags := []string{"tag1", "tag2", "tag3"}
	if len(cfg.Tags) != len(expectedTags) {
		t.Fatalf("Tags length = %d; want %d", len(cfg.Tags), len(expectedTags))
	}
	for i, tag := range cfg.Tags {
		if tag != expectedTags[i] {
			t.Errorf("Tags[%d] = %q; want %q", i, tag, expectedTags[i])
		}
	}

	// Verify int slice
	expectedPorts := []int{3000, 4000, 5000}
	if len(cfg.Ports) != len(expectedPorts) {
		t.Fatalf("Ports length = %d; want %d", len(cfg.Ports), len(expectedPorts))
	}
	for i, port := range cfg.Ports {
		if port != expectedPorts[i] {
			t.Errorf("Ports[%d] = %d; want %d", i, port, expectedPorts[i])
		}
	}

	// Verify float slice
	expectedRatios := []float64{1.5, 2.5, 3.5}
	if len(cfg.Ratios) != len(expectedRatios) {
		t.Fatalf("Ratios length = %d; want %d", len(cfg.Ratios), len(expectedRatios))
	}
	for i, ratio := range cfg.Ratios {
		if ratio != expectedRatios[i] {
			t.Errorf("Ratios[%d] = %f; want %f", i, ratio, expectedRatios[i])
		}
	}

	// Verify bool slice
	expectedFeatures := []bool{false, true, false}
	if len(cfg.Features) != len(expectedFeatures) {
		t.Fatalf("Features length = %d; want %d", len(cfg.Features), len(expectedFeatures))
	}
	for i, feature := range cfg.Features {
		if feature != expectedFeatures[i] {
			t.Errorf("Features[%d] = %v; want %v", i, feature, expectedFeatures[i])
		}
	}

	// Verify secret slice
	expectedSecrets := []string{"dotenv_secret1", "dotenv_secret2", "dotenv_secret3"}
	if len(cfg.Secrets) != len(expectedSecrets) {
		t.Fatalf("Secrets length = %d; want %d", len(cfg.Secrets), len(expectedSecrets))
	}
	for i, secret := range cfg.Secrets {
		if secret != expectedSecrets[i] {
			t.Errorf("Secrets[%d] = %q; want %q", i, secret, expectedSecrets[i])
		}
	}
}

// Test missing .env file (should not fail)
func TestDotenvMissingFile(t *testing.T) {
	type Config struct {
		Value string `env:"MISSING_FILE_VALUE" default:"default"`
	}

	cfg, err := LoadWithDotenv(Config{}, "/non/existent/path/.env")

	// Should not fail when .env file doesn't exist
	if err != nil {
		t.Fatalf("LoadWithDotenv should not fail for missing .env file: %v", err)
	}

	// Should use default value
	if cfg.Value != "default" {
		t.Errorf("Value = %q; want %q (should use default when .env missing)", cfg.Value, "default")
	}
}

// Test .env file with malformed content (should still work for valid lines)
func TestDotenvMalformedFile(t *testing.T) {
	type Config struct {
		ValidVar string `env:"VALID_VAR" default:"default"`
		Another  string `env:"ANOTHER_VAR" default:"default2"`
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .env file with some malformed lines
	envFile := filepath.Join(tempDir, ".env")
	envContent := `# This is a comment
VALID_VAR=valid_value
ANOTHER_VAR=another_value
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	cfg, err := LoadWithDotenv(Config{}, envFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv failed: %v", err)
	}

	// Valid variables should still be loaded
	if cfg.ValidVar != "valid_value" {
		t.Errorf("ValidVar = %q; want %q", cfg.ValidVar, "valid_value")
	}
	if cfg.Another != "another_value" {
		t.Errorf("Another = %q; want %q", cfg.Another, "another_value")
	}
}

// Test .env file with required fields
func TestDotenvWithRequiredFields(t *testing.T) {
	type Config struct {
		Required    string `env:"DOTENV_REQUIRED" required:"true"`
		NotRequired string `env:"DOTENV_NOT_REQUIRED" default:"default"`
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gonfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test 1: .env file with required field present
	envFile := filepath.Join(tempDir, ".env")
	envContent := `DOTENV_REQUIRED=required_value
DOTENV_NOT_REQUIRED=optional_value
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	cfg, err := LoadWithDotenv(Config{}, envFile)
	if err != nil {
		t.Fatalf("LoadWithDotenv should succeed when required field is in .env: %v", err)
	}

	if cfg.Required != "required_value" {
		t.Errorf("Required = %q; want %q", cfg.Required, "required_value")
	}

	// Clear environment variables before second test
	os.Unsetenv("DOTENV_REQUIRED")
	os.Unsetenv("DOTENV_NOT_REQUIRED")

	// Test 2: .env file missing required field
	envFile2 := filepath.Join(tempDir, "missing_required.env")
	envContent2 := `DOTENV_NOT_REQUIRED=optional_value
`
	if err := os.WriteFile(envFile2, []byte(envContent2), 0644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	_, err = LoadWithDotenv(Config{}, envFile2)
	if err == nil {
		t.Fatal("LoadWithDotenv should fail when required field is missing")
	}

	expectedErr := "required env \"DOTENV_REQUIRED\" missing"
	if err.Error() != expectedErr {
		t.Errorf("Error = %q; want %q", err.Error(), expectedErr)
	}
}
