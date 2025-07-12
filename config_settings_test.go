package getconfig

import (
	"net/url"
	"testing"
	"time"
)

func TestSettings(t *testing.T) {
	type NestedConfig struct {
		Host string `env:"DB_HOST" default:"localhost"`
		Port int    `env:"DB_PORT" default:"5432"`
	}

	type TestConfig struct {
		AppName     string        `env:"APP_NAME" default:"myapp"`
		APIKey      string        `secret:"API_KEY" required:"true"`
		Debug       bool          `env:"DEBUG" default:"false"`
		Timeout     time.Duration `env:"TIMEOUT" default:"30s"`
		DatabaseURL url.URL       `env:"DATABASE_URL"`
		Database    NestedConfig
		Optional    *NestedConfig `env:"OPTIONAL_DB"`
	}

	settings := Settings(TestConfig{})

	// Verify we got all expected fields including nested ones
	expectedFields := []string{"AppName", "APIKey", "Debug", "Timeout", "DatabaseURL", "Database.Host", "Database.Port", "Optional.Host", "Optional.Port"}
	if len(settings) != len(expectedFields) {
		t.Errorf("Expected %d settings, got %d", len(expectedFields), len(settings))
		t.Logf("Found settings:")
		for i, setting := range settings {
			t.Logf("  %d: %s", i, setting.Path)
		}
	}

	// Create a map for easier lookup
	settingsMap := make(map[string]FieldSetting)
	for _, setting := range settings {
		settingsMap[setting.Path] = setting
	}

	// Test specific field properties including nested fields
	tests := []struct {
		path       string
		envVar     string
		required   bool
		secret     bool
		hasDefault bool
	}{
		{"AppName", "APP_NAME", false, false, true},
		{"APIKey", "API_KEY", true, true, false},
		{"Debug", "DEBUG", false, false, true},
		{"Timeout", "TIMEOUT", false, false, true},
		{"DatabaseURL", "DATABASE_URL", false, false, false},
		{"Database.Host", "DB_HOST", false, false, true},
		{"Database.Port", "DB_PORT", false, false, true},
		{"Optional.Host", "DB_HOST", false, false, true},
		{"Optional.Port", "DB_PORT", false, false, true},
	}

	for _, test := range tests {
		setting, exists := settingsMap[test.path]
		if !exists {
			t.Errorf("Expected setting %s not found", test.path)
			continue
		}

		if setting.EnvVar != test.envVar {
			t.Errorf("Field %s: expected EnvVar %s, got %s", test.path, test.envVar, setting.EnvVar)
		}

		if setting.Required != test.required {
			t.Errorf("Field %s: expected Required %v, got %v", test.path, test.required, setting.Required)
		}

		if setting.Secret != test.secret {
			t.Errorf("Field %s: expected Secret %v, got %v", test.path, test.secret, setting.Secret)
		}

		hasDefault := setting.Default != ""
		if hasDefault != test.hasDefault {
			t.Errorf("Field %s: expected hasDefault %v, got %v (default: %q)", test.path, test.hasDefault, hasDefault, setting.Default)
		}
	}
}

func TestFilterSettings(t *testing.T) {
	type TestConfig struct {
		Public    string `env:"PUBLIC"`
		Secret1   string `secret:"SECRET1" required:"true"`
		Secret2   string `secret:"SECRET2"`
		Required1 string `env:"REQUIRED1" required:"true"`
		Required2 string `env:"REQUIRED2" required:"true"`
	}

	allSettings := Settings(TestConfig{})

	// Test filtering by predicate
	secretSettings := FilterSettings(allSettings, func(s FieldSetting) bool {
		return s.Secret
	})

	if len(secretSettings) != 2 {
		t.Errorf("Expected 2 secret settings, got %d", len(secretSettings))
	}

	requiredSettings := FilterSettings(allSettings, func(s FieldSetting) bool {
		return s.Required
	})

	if len(requiredSettings) != 3 { // Secret1, Required1, Required2
		t.Errorf("Expected 3 required settings, got %d", len(requiredSettings))
	}
}

func TestSecretFields(t *testing.T) {
	type TestConfig struct {
		Public  string `env:"PUBLIC"`
		Secret1 string `secret:"SECRET1"`
		Secret2 string `secret:"SECRET2"`
	}

	secrets := SecretFields(TestConfig{})

	if len(secrets) != 2 {
		t.Errorf("Expected 2 secret fields, got %d", len(secrets))
	}

	for _, secret := range secrets {
		if !secret.Secret {
			t.Errorf("Field %s should be marked as secret", secret.Path)
		}
	}
}

func TestSettingsRequiredFields(t *testing.T) {
	type TestConfig struct {
		Optional  string `env:"OPTIONAL"`
		Required1 string `env:"REQUIRED1" required:"true"`
		Required2 string `secret:"REQUIRED2" required:"true"`
	}

	required := RequiredFields(TestConfig{})

	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}

	for _, req := range required {
		if !req.Required {
			t.Errorf("Field %s should be marked as required", req.Path)
		}
	}
}

func TestSettingsWithPointerStruct(t *testing.T) {
	type NestedConfig struct {
		Host string `env:"HOST" default:"localhost"`
	}

	type TestConfig struct {
		Database *NestedConfig
	}

	settings := Settings(TestConfig{})

	// Should find the nested struct field even when nil
	found := false
	for _, setting := range settings {
		if setting.Path == "Database.Host" {
			found = true
			if setting.EnvVar != "HOST" {
				t.Errorf("Expected EnvVar HOST, got %s", setting.EnvVar)
			}
			break
		}
	}

	if !found {
		t.Error("Should find nested struct field in pointer struct")
	}
}

func TestSettingsNonStruct(t *testing.T) {
	// Test with non-struct type
	var notAStruct int
	settings := Settings(notAStruct)
	if settings != nil {
		t.Error("Settings should return nil for non-struct types")
	}

	// Test with pointer to non-struct
	settings = Settings(&notAStruct)
	if settings != nil {
		t.Error("Settings should return nil for pointer to non-struct types")
	}
}

func TestSettingsTypeNames(t *testing.T) {
	type TestConfig struct {
		StringField   string        `env:"STRING"`
		IntField      int           `env:"INT"`
		SliceField    []string      `env:"SLICE"`
		DurationField time.Duration `env:"DURATION"`
		URLField      url.URL       `env:"URL"`
	}

	settings := Settings(TestConfig{})

	typeMap := make(map[string]string)
	for _, setting := range settings {
		typeMap[setting.FieldName] = setting.Type
	}

	expectedTypes := map[string]string{
		"StringField":   "string",
		"IntField":      "int",
		"SliceField":    "[]string",
		"DurationField": "time.Duration",
		"URLField":      "url.URL", // The actual output is url.URL, not net/url.URL
	}

	for field, expectedType := range expectedTypes {
		if actualType, exists := typeMap[field]; !exists {
			t.Errorf("Field %s not found", field)
		} else if actualType != expectedType {
			t.Errorf("Field %s: expected type %s, got %s", field, expectedType, actualType)
		}
	}
}
