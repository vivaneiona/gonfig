package getconfig

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prettyTestConfig is used to test PrettyString behavior
type prettyTestConfig struct {
	Field1      string `env:"FIELD1"`
	SecretField string `secret:"SECRET_FIELD"`
	NoTagField  string
}

func TestMask(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"a", "*"},
		{"ab", "**"},
		{"abc", "***"},
		{"abcd", "abc*"},
		{"abcdef", "abc***"},
	}
	for _, c := range cases {
		got := mask(c.input)
		if got != c.want {
			t.Errorf("mask(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}

func TestPrettyString(t *testing.T) {
	cfg := &prettyTestConfig{
		Field1:      "value",
		SecretField: "abcdef",
		NoTagField:  "visible",
	}
	out := PrettyString(cfg)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse PrettyString output: %v", err)
	}

	if result["FIELD1"] != "value" {
		t.Errorf("FIELD1 = %v; want \"value\"", result["FIELD1"])
	}

	if result["SECRET_FIELD"] != "abc***" {
		t.Errorf("SECRET_FIELD = %v; want \"abc***\"", result["SECRET_FIELD"])
	}

	if result["NoTagField"] != "visible" {
		t.Errorf("NoTagField = %v; want \"visible\"", result["NoTagField"])
	}
}

// loadTestConfig includes bool, int, and float fields
type loadTestConfig struct {
	Value1   string  `env:"VALUE1" default:"def1"`
	Value2   string  `secret:"VALUE2" default:"def2"`
	Flag     bool    `env:"FLAG"   default:"1"`
	IntVal   int     `env:"INTVAL" default:"7"`
	FloatVal float64 `env:"FLOATVAL" default:"3.14"`
	NoEnv    string
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name      string
		envs      map[string]string
		want1     string
		want2     string
		wantFlag  bool
		wantInt   int
		wantFloat float64
	}{
		{
			name:      "no env set",
			envs:      nil,
			want1:     "def1",
			want2:     "def2",
			wantFlag:  true,
			wantInt:   7,
			wantFloat: 3.14,
		},
		{
			name:      "env override",
			envs:      map[string]string{"VALUE1": "env1", "VALUE2": "env2", "FLAG": "false", "INTVAL": "42", "FLOATVAL": "2.718"},
			want1:     "env1",
			want2:     "env2",
			wantFlag:  false,
			wantInt:   42,
			wantFloat: 2.718,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			cfg, err := Load(loadTestConfig{})
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			if cfg.Value1 != tc.want1 {
				t.Errorf("Value1 = %q; want %q", cfg.Value1, tc.want1)
			}
			if cfg.Value2 != tc.want2 {
				t.Errorf("Value2 = %q; want %q", cfg.Value2, tc.want2)
			}
			if cfg.Flag != tc.wantFlag {
				t.Errorf("Flag = %v; want %v", cfg.Flag, tc.wantFlag)
			}
			if cfg.IntVal != tc.wantInt {
				t.Errorf("IntVal = %d; want %d", cfg.IntVal, tc.wantInt)
			}
			if cfg.FloatVal != tc.wantFloat {
				t.Errorf("FloatVal = %v; want %v", cfg.FloatVal, tc.wantFloat)
			}
			if cfg.NoEnv != "" {
				t.Errorf("NoEnv = %q; want empty string", cfg.NoEnv)
			}
		})
	}
}

// Test nested struct support
func TestNestedStruct(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"DB_HOST" default:"localhost"`
		Port     int    `env:"DB_PORT" default:"5432"`
		Password string `secret:"DB_PASSWORD" default:"secret"`
	}

	type AppConfig struct {
		Name string `env:"APP_NAME" default:"myapp"`
		Port int    `env:"PORT" default:"8080"`
		DB   DatabaseConfig
	}

	// Set some environment variables
	t.Setenv("APP_NAME", "testapp")
	t.Setenv("DB_HOST", "dbserver")
	t.Setenv("DB_PORT", "3306")

	cfg, err := Load(AppConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check top-level fields
	if cfg.Name != "testapp" {
		t.Errorf("Name = %q; want %q", cfg.Name, "testapp")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d; want %d", cfg.Port, 8080)
	}

	// Check nested struct fields
	if cfg.DB.Host != "dbserver" {
		t.Errorf("DB.Host = %q; want %q", cfg.DB.Host, "dbserver")
	}
	if cfg.DB.Port != 3306 {
		t.Errorf("DB.Port = %d; want %d", cfg.DB.Port, 3306)
	}
	if cfg.DB.Password != "secret" {
		t.Errorf("DB.Password = %q; want %q", cfg.DB.Password, "secret")
	}
}

// Test nested pointer struct support
func TestNestedPointerStruct(t *testing.T) {
	type RedisConfig struct {
		Host string `env:"REDIS_HOST" default:"localhost"`
		Port int    `env:"REDIS_PORT" default:"6379"`
	}

	type AppConfig struct {
		Name  string `env:"APP_NAME" default:"myapp"`
		Redis *RedisConfig
	}

	// Set some environment variables
	t.Setenv("REDIS_HOST", "redis-server")

	cfg, err := Load(AppConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that pointer was allocated
	if cfg.Redis == nil {
		t.Fatal("Redis config should not be nil")
	}

	// Check nested fields
	if cfg.Redis.Host != "redis-server" {
		t.Errorf("Redis.Host = %q; want %q", cfg.Redis.Host, "redis-server")
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d; want %d", cfg.Redis.Port, 6379)
	}
}

// Test PrettyString with nested structs
func TestPrettyStringNested(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"DB_HOST"`
		Password string `secret:"DB_PASSWORD"`
	}

	type AppConfig struct {
		Name string `env:"APP_NAME"`
		DB   DatabaseConfig
	}

	cfg := &AppConfig{
		Name: "testapp",
		DB: DatabaseConfig{
			Host:     "dbserver",
			Password: "supersecret",
		},
	}

	out := PrettyString(cfg)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse PrettyString output: %v", err)
	}

	// Check top-level field
	if result["APP_NAME"] != "testapp" {
		t.Errorf("APP_NAME = %v; want %q", result["APP_NAME"], "testapp")
	}

	// Check nested struct
	dbConfig, ok := result["DB"].(map[string]interface{})
	if !ok {
		t.Fatalf("DB should be a nested object, got %T", result["DB"])
	}

	if dbConfig["DB_HOST"] != "dbserver" {
		t.Errorf("DB.DB_HOST = %v; want %q", dbConfig["DB_HOST"], "dbserver")
	}

	// Check that secret is masked
	if dbConfig["DB_PASSWORD"] != "sup********" {
		t.Errorf("DB.DB_PASSWORD = %v; want %q", dbConfig["DB_PASSWORD"], "sup********")
	}
}

// Test required fields
func TestRequiredFields(t *testing.T) {
	type Config struct {
		Required    string `env:"REQUIRED_FIELD" required:"true"`
		NotRequired string `env:"NOT_REQUIRED"`
	}

	_, err := Load(Config{})
	if err == nil {
		t.Fatal("Expected error for missing required field")
	}

	expectedErr := "required env \"REQUIRED_FIELD\" missing"
	if err.Error() != expectedErr {
		t.Errorf("Error = %q; want %q", err.Error(), expectedErr)
	}
}

// Test slice support
func TestSliceSupport(t *testing.T) {
	type Config struct {
		Strings []string `env:"STRINGS" default:"a,b,c"`
		Numbers []int    `env:"NUMBERS" default:"1,2,3"`
	}

	t.Setenv("STRINGS", "x,y,z")

	cfg, err := Load(Config{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedStrings := []string{"x", "y", "z"}
	if len(cfg.Strings) != len(expectedStrings) {
		t.Fatalf("Strings length = %d; want %d", len(cfg.Strings), len(expectedStrings))
	}
	for i, s := range cfg.Strings {
		if s != expectedStrings[i] {
			t.Errorf("Strings[%d] = %q; want %q", i, s, expectedStrings[i])
		}
	}

	expectedNumbers := []int{1, 2, 3}
	if len(cfg.Numbers) != len(expectedNumbers) {
		t.Fatalf("Numbers length = %d; want %d", len(cfg.Numbers), len(expectedNumbers))
	}
	for i, n := range cfg.Numbers {
		if n != expectedNumbers[i] {
			t.Errorf("Numbers[%d] = %d; want %d", i, n, expectedNumbers[i])
		}
	}
}

// Test comprehensive CSV values
func TestCSVValues(t *testing.T) {
	type CSVTestConfig struct {
		CsvStrings []string  `env:"CSVSTRINGS" default:"foo1,foo2,foo3"`
		CsvInts    []int     `env:"CSVINTS" default:"1,2,3"`
		CsvFloats  []float64 `env:"CSVFLOATS" default:"1.1,2.2,3.3"`
		CsvBools   []bool    `env:"CSVBOOLS" default:"true,false,true"`
		CsvSecrets []string  `secret:"CSVSECRETS" default:"secret1,secret2,secret3"`
	}

	// Test with environment variables override
	t.Setenv("CSVSTRINGS", "x,y,z")
	t.Setenv("CSVINTS", "10,20,30")
	t.Setenv("CSVFLOATS", "1.5,2.5,3.5")
	t.Setenv("CSVBOOLS", "false,true,false")
	t.Setenv("CSVSECRETS", "newsecret1,newsecret2,newsecret3")

	cfg, err := Load(CSVTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test string slice
	expectedStrings := []string{"x", "y", "z"}
	if len(cfg.CsvStrings) != len(expectedStrings) {
		t.Fatalf("CsvStrings length = %d; want %d", len(cfg.CsvStrings), len(expectedStrings))
	}
	for i, s := range cfg.CsvStrings {
		if s != expectedStrings[i] {
			t.Errorf("CsvStrings[%d] = %q; want %q", i, s, expectedStrings[i])
		}
	}

	// Test int slice
	expectedInts := []int{10, 20, 30}
	if len(cfg.CsvInts) != len(expectedInts) {
		t.Fatalf("CsvInts length = %d; want %d", len(cfg.CsvInts), len(expectedInts))
	}
	for i, n := range cfg.CsvInts {
		if n != expectedInts[i] {
			t.Errorf("CsvInts[%d] = %d; want %d", i, n, expectedInts[i])
		}
	}

	// Test float slice
	expectedFloats := []float64{1.5, 2.5, 3.5}
	if len(cfg.CsvFloats) != len(expectedFloats) {
		t.Fatalf("CsvFloats length = %d; want %d", len(cfg.CsvFloats), len(expectedFloats))
	}
	for i, f := range cfg.CsvFloats {
		if f != expectedFloats[i] {
			t.Errorf("CsvFloats[%d] = %f; want %f", i, f, expectedFloats[i])
		}
	}

	// Test bool slice
	expectedBools := []bool{false, true, false}
	if len(cfg.CsvBools) != len(expectedBools) {
		t.Fatalf("CsvBools length = %d; want %d", len(cfg.CsvBools), len(expectedBools))
	}
	for i, b := range cfg.CsvBools {
		if b != expectedBools[i] {
			t.Errorf("CsvBools[%d] = %v; want %v", i, b, expectedBools[i])
		}
	}

	// Test secret slice
	expectedSecrets := []string{"newsecret1", "newsecret2", "newsecret3"}
	if len(cfg.CsvSecrets) != len(expectedSecrets) {
		t.Fatalf("CsvSecrets length = %d; want %d", len(cfg.CsvSecrets), len(expectedSecrets))
	}
	for i, s := range cfg.CsvSecrets {
		if s != expectedSecrets[i] {
			t.Errorf("CsvSecrets[%d] = %q; want %q", i, s, expectedSecrets[i])
		}
	}
}

// Test CSV values with defaults (no env vars set)
func TestCSVValuesDefaults(t *testing.T) {
	type CSVTestConfig struct {
		CsvStrings []string  `env:"CSVSTRINGS_DEF" default:"foo1,foo2,foo3"`
		CsvInts    []int     `env:"CSVINTS_DEF" default:"1,2,3"`
		CsvFloats  []float64 `env:"CSVFLOATS_DEF" default:"1.1,2.2,3.3"`
		CsvBools   []bool    `env:"CSVBOOLS_DEF" default:"true,false,true"`
	}

	cfg, err := Load(CSVTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test default string slice
	expectedStrings := []string{"foo1", "foo2", "foo3"}
	if len(cfg.CsvStrings) != len(expectedStrings) {
		t.Fatalf("CsvStrings length = %d; want %d", len(cfg.CsvStrings), len(expectedStrings))
	}
	for i, s := range cfg.CsvStrings {
		if s != expectedStrings[i] {
			t.Errorf("CsvStrings[%d] = %q; want %q", i, s, expectedStrings[i])
		}
	}

	// Test default int slice
	expectedInts := []int{1, 2, 3}
	if len(cfg.CsvInts) != len(expectedInts) {
		t.Fatalf("CsvInts length = %d; want %d", len(cfg.CsvInts), len(expectedInts))
	}
	for i, n := range cfg.CsvInts {
		if n != expectedInts[i] {
			t.Errorf("CsvInts[%d] = %d; want %d", i, n, expectedInts[i])
		}
	}

	// Test default float slice
	expectedFloats := []float64{1.1, 2.2, 3.3}
	if len(cfg.CsvFloats) != len(expectedFloats) {
		t.Fatalf("CsvFloats length = %d; want %d", len(cfg.CsvFloats), len(expectedFloats))
	}
	for i, f := range cfg.CsvFloats {
		if f != expectedFloats[i] {
			t.Errorf("CsvFloats[%d] = %f; want %f", i, f, expectedFloats[i])
		}
	}

	// Test default bool slice
	expectedBools := []bool{true, false, true}
	if len(cfg.CsvBools) != len(expectedBools) {
		t.Fatalf("CsvBools length = %d; want %d", len(cfg.CsvBools), len(expectedBools))
	}
	for i, b := range cfg.CsvBools {
		if b != expectedBools[i] {
			t.Errorf("CsvBools[%d] = %v; want %v", i, b, expectedBools[i])
		}
	}
}

// Test PrettyString with CSV slices and secret masking
func TestPrettyStringCSVSlices(t *testing.T) {
	type CSVConfig struct {
		PublicStrings []string `env:"PUBLIC_STRINGS"`
		SecretStrings []string `secret:"SECRET_STRINGS"`
		Numbers       []int    `env:"NUMBERS"`
	}

	cfg := &CSVConfig{
		PublicStrings: []string{"public1", "public2", "public3"},
		SecretStrings: []string{"secret1", "secret2", "secret3"},
		Numbers:       []int{1, 2, 3},
	}

	out := PrettyString(cfg)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse PrettyString output: %v", err)
	}

	// Check public strings slice
	publicStrings, ok := result["PUBLIC_STRINGS"].([]interface{})
	if !ok {
		t.Fatalf("PUBLIC_STRINGS should be a slice, got %T", result["PUBLIC_STRINGS"])
	}
	expectedPublic := []string{"public1", "public2", "public3"}
	for i, v := range publicStrings {
		if v != expectedPublic[i] {
			t.Errorf("PUBLIC_STRINGS[%d] = %v; want %q", i, v, expectedPublic[i])
		}
	}

	// Check secret strings slice (should be masked)
	secretStrings, ok := result["SECRET_STRINGS"].([]interface{})
	if !ok {
		t.Fatalf("SECRET_STRINGS should be a slice, got %T", result["SECRET_STRINGS"])
	}
	expectedSecret := []string{"sec****", "sec****", "sec****"}
	for i, v := range secretStrings {
		if v != expectedSecret[i] {
			t.Errorf("SECRET_STRINGS[%d] = %v; want %q", i, v, expectedSecret[i])
		}
	}
}

// TestDefaultShouldNotOverrideExistingValues checks that default values do not override
// existing non-zero values in the configuration struct.
func TestDefaultShouldNotOverrideExistingValues(t *testing.T) {
	type Config struct {
		Value string `env:"VALUE" default:"default-value"`
	}

	// Pre-set a value in the struct
	cfg := &Config{Value: "existing-value"}

	// No environment variable is set, so the default would normally apply
	_, err := Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// The existing value should be preserved
	if cfg.Value != "existing-value" {
		t.Errorf("Value = %q; want %q", cfg.Value, "existing-value")
	}
}

// TestNestedStructDefaultShouldNotOverrideExistingValues checks that default values
// do not override existing non-zero values in nested structs.
func TestNestedStructDefaultShouldNotOverrideExistingValues(t *testing.T) {
	type DatabaseConfig struct {
		Host string `env:"DB_HOST" default:"default-host"`
		Port int    `env:"DB_PORT" default:"5432"`
	}

	type ServerConfig struct {
		Name string `env:"SERVER_NAME" default:"default-server"`
	}

	type AppConfig struct {
		Name   string `env:"APP_NAME" default:"default-app"`
		Server *ServerConfig
		DB     DatabaseConfig
	}

	// Pre-set values in the nested struct
	cfg := &AppConfig{
		Name: "app",
		Server: &ServerConfig{
			Name: "existing-server",
		},
		DB: DatabaseConfig{
			Host: "existing-host",
			Port: 9999,
		},
	}

	_, err := Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that existing values in nested structs are preserved
	if cfg.DB.Host != "existing-host" {
		t.Errorf("DB.Host = %q; want %q", cfg.DB.Host, "existing-host")
	}
	if cfg.DB.Port != 9999 {
		t.Errorf("DB.Port = %d; want %d", cfg.DB.Port, 9999)
	}
	if cfg.Server.Name != "existing-server" {
		t.Errorf("Server.Name = %q; want %q", cfg.Server.Name, "existing-server")
	}
}

// TestNestedStructMixedValues tests loading with a mix of env vars, defaults, and zero values
func TestNestedStructMixedValues(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"DB_HOST" default:"localhost"`
		Port     int    `env:"DB_PORT" default:"5432"`
		Username string `env:"DB_USER"` // No default
	}

	type AppConfig struct {
		Name string `env:"APP_NAME" default:"myapp"`
		DB   DatabaseConfig
	}

	// Set only one of the nested env vars
	t.Setenv("DB_HOST", "remote-db")

	cfg, err := Load(AppConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that the mix of values is loaded correctly
	if cfg.Name != "myapp" {
		t.Errorf("Name = %q; want %q", cfg.Name, "myapp")
	}
	if cfg.DB.Host != "remote-db" {
		t.Errorf("DB.Host = %q; want %q", cfg.DB.Host, "remote-db")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d; want %d", cfg.DB.Port, 5432)
	}
	if cfg.DB.Username != "" {
		t.Errorf("DB.Username should be empty, got %q", cfg.DB.Username)
	}
}

// TestNestedStructWithEnvOverrides tests that environment variables correctly
// override default values in nested structs.
func TestNestedStructWithEnvOverrides(t *testing.T) {
	type DatabaseConfig struct {
		Host string `env:"DB_HOST" default:"localhost"`
		Port int    `env:"DB_PORT" default:"5432"`
	}

	type AppConfig struct {
		Name string `env:"APP_NAME" default:"myapp"`
		DB   DatabaseConfig
	}

	// Override all relevant environment variables
	t.Setenv("APP_NAME", "overridden-app")
	t.Setenv("DB_HOST", "overridden-host")
	t.Setenv("DB_PORT", "9999")

	cfg, err := Load(AppConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that all values were overridden
	if cfg.Name != "overridden-app" {
		t.Errorf("Name = %q; want %q", cfg.Name, "overridden-app")
	}
	if cfg.DB.Host != "overridden-host" {
		t.Errorf("DB.Host = %q; want %q", cfg.DB.Host, "overridden-host")
	}
	if cfg.DB.Port != 9999 {
		t.Errorf("DB.Port = %d; want %d", cfg.DB.Port, 9999)
	}
}

// Test environment variables override everything, including defaults and pre-set values
func TestEnvironmentVariablesOverrideEverything(t *testing.T) {
	type DatabaseConfig struct {
		Host     string `env:"DB_HOST" default:"default-host"`
		Port     int    `env:"DB_PORT" default:"5432"`
		Password string `secret:"DB_PASSWORD" default:"default-password"`
	}

	type ServerConfig struct {
		Name string `env:"SERVER_NAME" default:"default-server"`
	}

	type AppConfig struct {
		Name   string `env:"APP_NAME" default:"default-app"`
		Server *ServerConfig
		DB     DatabaseConfig
	}

	// Pre-set some values
	cfg := &AppConfig{
		Name: "pre-set-app",
		Server: &ServerConfig{
			Name: "pre-set-server",
		},
		DB: DatabaseConfig{
			Host:     "pre-set-host",
			Port:     1111,
			Password: "pre-set-password",
		},
	}

	// Set environment variables that should override everything
	t.Setenv("APP_NAME", "env-app")
	t.Setenv("SERVER_NAME", "env-server")
	t.Setenv("DB_HOST", "env-host")
	t.Setenv("DB_PORT", "9999")
	t.Setenv("DB_PASSWORD", "env-password")

	_, err := Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that environment variables took precedence
	if cfg.Name != "env-app" {
		t.Errorf("App Name = %q; want %q", cfg.Name, "env-app")
	}
	if cfg.Server.Name != "env-server" {
		t.Errorf("Server Name = %q; want %q", cfg.Server.Name, "env-server")
	}
	if cfg.DB.Host != "env-host" {
		t.Errorf("DB Host = %q; want %q", cfg.DB.Host, "env-host")
	}
	if cfg.DB.Port != 9999 {
		t.Errorf("DB Port = %d; want %d", cfg.DB.Port, 9999)
	}
	if cfg.DB.Password != "env-password" {
		t.Errorf("DB Password = %q; want %q", cfg.DB.Password, "env-password")
	}
}

func TestNestedPointerStruct_DefaultAllocation(t *testing.T) {
	os.Clearenv()
	type Redis struct {
		Addr string `env:"REDIS_ADDR" default:"localhost:6379"`
	}
	type Cfg struct {
		Redis *Redis
	}
	var c Cfg
	_, err := Load(&c)
	require.NoError(t, err)
	require.NotNil(t, c.Redis, "pointer should be auto-allocated")
	assert.Equal(t, "localhost:6379", c.Redis.Addr)
}
