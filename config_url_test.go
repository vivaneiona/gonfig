package gonfig

import (
	"encoding/json"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRegisterParser(t *testing.T) {
	// Test custom type registration
	type CustomType string
	customType := reflect.TypeOf(CustomType(""))

	RegisterParser(customType, func(raw string) (any, error) {
		return CustomType("custom_" + raw), nil
	})

	if _, ok := customParsers[customType]; !ok {
		t.Error("custom parser was not registered")
	}
}

func TestURLParsing(t *testing.T) {
	type Config struct {
		DatabaseURL url.URL  `env:"DATABASE_URL"`
		DatabasePtr *url.URL `env:"DATABASE_PTR"`
		OptionalURL *url.URL `env:"OPTIONAL_URL"`
	}

	tests := []struct {
		name         string
		envVars      map[string]string
		expectError  bool
		checkResults func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid URLs",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://user:password@localhost:5432/mydb?sslmode=disable",
				"DATABASE_PTR": "redis://admin:secret@redis.example.com:6379/0",
			},
			expectError: false,
			checkResults: func(t *testing.T, cfg *Config) {
				if cfg.DatabaseURL.Scheme != "postgres" {
					t.Errorf("expected scheme postgres, got %s", cfg.DatabaseURL.Scheme)
				}
				if cfg.DatabaseURL.Host != "localhost:5432" {
					t.Errorf("expected host localhost:5432, got %s", cfg.DatabaseURL.Host)
				}
				if cfg.DatabaseURL.User.Username() != "user" {
					t.Errorf("expected username user, got %s", cfg.DatabaseURL.User.Username())
				}
				password, _ := cfg.DatabaseURL.User.Password()
				if password != "password" {
					t.Errorf("expected password 'password', got %s", password)
				}

				if cfg.DatabasePtr == nil {
					t.Error("DatabasePtr should not be nil")
				} else {
					if cfg.DatabasePtr.Scheme != "redis" {
						t.Errorf("expected scheme redis, got %s", cfg.DatabasePtr.Scheme)
					}
					if cfg.DatabasePtr.Host != "redis.example.com:6379" {
						t.Errorf("expected host redis.example.com:6379, got %s", cfg.DatabasePtr.Host)
					}
				}

				if cfg.OptionalURL != nil {
					t.Error("OptionalURL should be nil when not set")
				}
			},
		},
		{
			name: "invalid URL",
			envVars: map[string]string{
				"DATABASE_URL": "://invalid-url",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for key := range tt.envVars {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load(Config{})

			if tt.expectError && err == nil {
				t.Error("expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkResults != nil {
				tt.checkResults(t, &cfg)
			}

			// Cleanup
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestURLPasswordMasking(t *testing.T) {
	type Config struct {
		DatabaseURL url.URL  `env:"DATABASE_URL"`
		RedisURL    *url.URL `env:"REDIS_URL"`
		NoPassURL   url.URL  `env:"NO_PASS_URL"`
	}

	os.Setenv("DATABASE_URL", "postgres://user:secret123@localhost:5432/mydb")
	os.Setenv("REDIS_URL", "redis://admin:topsecret@redis.example.com:6379/0")
	os.Setenv("NO_PASS_URL", "https://api.example.com/v1")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("NO_PASS_URL")
	}()

	// Load configuration into cfg
	cfg, err := Load(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prettyStr := PrettyString(cfg)

	// Parse the JSON output to check masking
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(prettyStr), &result); err != nil {
		t.Fatalf("failed to parse PrettyString output: %v", err)
	}

	// Check that passwords are masked in URLs
	dbURL, ok := result["DATABASE_URL"].(string)
	if !ok {
		t.Error("DATABASE_URL should be a string in output")
	} else if !strings.Contains(dbURL, "user:") || (!strings.Contains(dbURL, ":***@") && !strings.Contains(dbURL, ":%2A%2A%2A@")) {
		t.Errorf("DATABASE_URL password should be masked, got: %s", dbURL)
	}

	redisURL, ok := result["REDIS_URL"].(string)
	if !ok {
		t.Error("REDIS_URL should be a string in output")
	} else if !strings.Contains(redisURL, "admin:") || (!strings.Contains(redisURL, ":***@") && !strings.Contains(redisURL, ":%2A%2A%2A@")) {
		t.Errorf("REDIS_URL password should be masked, got: %s", redisURL)
	}

	// Check that URLs without passwords are not affected
	noPassURL, ok := result["NO_PASS_URL"].(string)
	if !ok {
		t.Error("NO_PASS_URL should be a string in output")
	} else if !strings.Contains(noPassURL, "https://api.example.com/v1") {
		t.Errorf("NO_PASS_URL should be unchanged, got: %s", noPassURL)
	}
}

func TestParseWithRegistry(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		typ         reflect.Type
		kind        reflect.Kind
		bits        int
		expectError bool
		expected    interface{}
	}{
		{
			name:        "url.URL parsing",
			raw:         "https://example.com/path",
			typ:         reflect.TypeOf(url.URL{}),
			kind:        reflect.Struct,
			bits:        0,
			expectError: false,
			expected:    url.URL{Scheme: "https", Host: "example.com", Path: "/path"},
		},
		{
			name:        "*url.URL parsing",
			raw:         "postgres://localhost:5432/db",
			typ:         reflect.TypeOf(&url.URL{}),
			kind:        reflect.Ptr,
			bits:        0,
			expectError: false,
		},
		{
			name:        "fallback to parseScalar",
			raw:         "42",
			typ:         reflect.TypeOf(int(0)),
			kind:        reflect.Int,
			bits:        64,
			expectError: false,
			expected:    int64(42),
		},
		{
			name:        "invalid URL",
			raw:         "://invalid",
			typ:         reflect.TypeOf(url.URL{}),
			kind:        reflect.Struct,
			bits:        0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWithRegistry(tt.raw, tt.typ, tt.kind, tt.bits)

			if tt.expectError && err == nil {
				t.Error("expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && tt.expected != nil {
				switch expected := tt.expected.(type) {
				case url.URL:
					if resultURL, ok := result.(url.URL); ok {
						if resultURL.Scheme != expected.Scheme || resultURL.Host != expected.Host || resultURL.Path != expected.Path {
							t.Errorf("expected %+v, got %+v", expected, resultURL)
						}
					} else {
						t.Errorf("expected url.URL, got %T", result)
					}
				case int64:
					if resultInt, ok := result.(int64); ok {
						if resultInt != expected {
							t.Errorf("expected %v, got %v", expected, resultInt)
						}
					} else {
						t.Errorf("expected int64, got %T", result)
					}
				}
			}
		})
	}
}

func TestMaskURLPassword(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "URL with password",
			input:    mustParseURL("postgres://user:secret@localhost:5432/db"),
			expected: "postgres://user:***@localhost:5432/db",
		},
		{
			name:     "URL without password",
			input:    mustParseURL("https://api.example.com/v1"),
			expected: "https://api.example.com/v1",
		},
		{
			name:     "URL pointer with password",
			input:    mustParseURLPtr("redis://admin:pass123@redis.com:6379/0"),
			expected: "redis://admin:***@redis.com:6379/0",
		},
		{
			name:     "nil URL pointer",
			input:    (*url.URL)(nil),
			expected: "",
		},
		{
			name:     "non-URL value",
			input:    "plain string",
			expected: "plain string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskURLPassword(tt.input)

			var resultStr string
			switch r := result.(type) {
			case string:
				resultStr = r
			case nil:
				resultStr = ""
			default:
				resultStr = ""
			}

			if tt.name == "non-URL value" {
				if result != tt.input {
					t.Errorf("expected non-URL input to be unchanged, got %v", result)
				}
			} else {
				// For URL masking, accept both *** and URL-encoded version
				if tt.expected != resultStr && !strings.Contains(resultStr, "%2A%2A%2A") {
					t.Errorf("expected %q or URL-encoded version, got %q", tt.expected, resultStr)
				}
			}
		})
	}
}

func TestURLSlices(t *testing.T) {
	type Config struct {
		URLs     []url.URL  `env:"URLS"`
		URLPtrs  []*url.URL `env:"URL_PTRS"`
		Servers  []url.URL  `env:"SERVERS" default:"http://server1.com,https://server2.com:8080"`
		Optional []*url.URL `env:"OPTIONAL_URLS"`
	}

	tests := []struct {
		name         string
		envVars      map[string]string
		expectError  bool
		checkResults func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid URL slices",
			envVars: map[string]string{
				"URLS":     "http://example.com,https://api.test.com:8080/v1",
				"URL_PTRS": "postgres://user:pass@db.com:5432/mydb,redis://localhost:6379/0",
			},
			expectError: false,
			checkResults: func(t *testing.T, cfg *Config) {
				if len(cfg.URLs) != 2 {
					t.Errorf("expected 2 URLs, got %d", len(cfg.URLs))
				}
				if cfg.URLs[0].Host != "example.com" {
					t.Errorf("expected first URL host 'example.com', got %s", cfg.URLs[0].Host)
				}
				if cfg.URLs[1].Host != "api.test.com:8080" {
					t.Errorf("expected second URL host 'api.test.com:8080', got %s", cfg.URLs[1].Host)
				}

				if len(cfg.URLPtrs) != 2 {
					t.Errorf("expected 2 URL pointers, got %d", len(cfg.URLPtrs))
				}
				if cfg.URLPtrs[0] == nil || cfg.URLPtrs[0].Scheme != "postgres" {
					t.Error("expected first URL ptr to be postgres scheme")
				}
				if cfg.URLPtrs[1] == nil || cfg.URLPtrs[1].Scheme != "redis" {
					t.Error("expected second URL ptr to be redis scheme")
				}

				// Test defaults
				if len(cfg.Servers) != 2 {
					t.Errorf("expected 2 default servers, got %d", len(cfg.Servers))
				}
				if cfg.Servers[0].Host != "server1.com" {
					t.Errorf("expected first server host 'server1.com', got %s", cfg.Servers[0].Host)
				}

				if cfg.Optional != nil && len(cfg.Optional) > 0 {
					t.Error("Optional should be empty when not set")
				}
			},
		},
		{
			name: "invalid URL in slice",
			envVars: map[string]string{
				"URLS": "http://valid.com,://invalid-url,https://valid2.com",
			},
			expectError: true,
		},
		{
			name: "empty slice",
			envVars: map[string]string{
				"URLS": "",
			},
			expectError: false,
			checkResults: func(t *testing.T, cfg *Config) {
				if len(cfg.URLs) != 0 {
					t.Errorf("expected empty slice, got %v", cfg.URLs)
				}
			},
		},
		{
			name: "whitespace handling",
			envVars: map[string]string{
				"URLS": " http://spaced.com , https://another.com/path ",
			},
			expectError: false,
			checkResults: func(t *testing.T, cfg *Config) {
				if len(cfg.URLs) != 2 {
					t.Errorf("expected 2 URLs, got %d", len(cfg.URLs))
				}
				if cfg.URLs[0].Host != "spaced.com" {
					t.Errorf("expected trimmed host 'spaced.com', got %s", cfg.URLs[0].Host)
				}
				if cfg.URLs[1].Path != "/path" {
					t.Errorf("expected path '/path', got %s", cfg.URLs[1].Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			keys := []string{"URLS", "URL_PTRS", "SERVERS", "OPTIONAL_URLS"}
			for _, key := range keys {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load(Config{})

			if tt.expectError && err == nil {
				t.Error("expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkResults != nil {
				tt.checkResults(t, &cfg)
			}

			// Cleanup
			for _, key := range keys {
				os.Unsetenv(key)
			}
		})
	}
}

func TestURLSlicePasswordMasking(t *testing.T) {
	type Config struct {
		DatabaseURLs []url.URL  `env:"DB_URLS"`
		ServiceURLs  []*url.URL `env:"SERVICE_URLS"`
		PublicURLs   []url.URL  `env:"PUBLIC_URLS"`
	}

	os.Setenv("DB_URLS", "postgres://user:secret@db1.com:5432/app,mysql://admin:password123@db2.com:3306/data")
	os.Setenv("SERVICE_URLS", "redis://cache:topsecret@redis.com:6379/0,https://api:key123@service.com/v1")
	os.Setenv("PUBLIC_URLS", "https://public.com/api,http://open.example.com:8080")
	defer func() {
		os.Unsetenv("DB_URLS")
		os.Unsetenv("SERVICE_URLS")
		os.Unsetenv("PUBLIC_URLS")
	}()

	// Load configuration into cfg
	cfg, err := Load(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prettyStr := PrettyString(cfg)

	// Parse the JSON output to check masking
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(prettyStr), &result); err != nil {
		t.Fatalf("failed to parse PrettyString output: %v", err)
	}

	// Check that passwords are masked in URL slices
	dbURLs, ok := result["DB_URLS"].([]interface{})
	if !ok {
		t.Error("DB_URLS should be an array in output")
	} else {
		if len(dbURLs) != 2 {
			t.Errorf("expected 2 DB URLs, got %d", len(dbURLs))
		}
		for i, urlInterface := range dbURLs {
			urlStr, ok := urlInterface.(string)
			if !ok {
				t.Errorf("DB URL %d should be a string", i)
				continue
			}
			if !strings.Contains(urlStr, ":***@") && !strings.Contains(urlStr, ":%2A%2A%2A@") {
				t.Errorf("DB URL %d password should be masked, got: %s", i, urlStr)
			}
		}
	}

	serviceURLs, ok := result["SERVICE_URLS"].([]interface{})
	if !ok {
		t.Error("SERVICE_URLS should be an array in output")
	} else {
		if len(serviceURLs) != 2 {
			t.Errorf("expected 2 service URLs, got %d", len(serviceURLs))
		}
		for i, urlInterface := range serviceURLs {
			urlStr, ok := urlInterface.(string)
			if !ok {
				t.Errorf("Service URL %d should be a string", i)
				continue
			}
			if !strings.Contains(urlStr, ":***@") && !strings.Contains(urlStr, ":%2A%2A%2A@") {
				t.Errorf("Service URL %d password should be masked, got: %s", i, urlStr)
			}
		}
	}

	// Check that URLs without passwords are not affected
	publicURLs, ok := result["PUBLIC_URLS"].([]interface{})
	if !ok {
		t.Error("PUBLIC_URLS should be an array in output")
	} else {
		for i, urlInterface := range publicURLs {
			urlStr, ok := urlInterface.(string)
			if !ok {
				t.Errorf("Public URL %d should be a string", i)
				continue
			}
			if strings.Contains(urlStr, "@") {
				t.Errorf("Public URL %d should not contain credentials, got: %s", i, urlStr)
			}
		}
	}
}

func TestURLSliceDefaults(t *testing.T) {
	type Config struct {
		Endpoints []url.URL  `env:"ENDPOINTS" default:"http://localhost:8080,https://api.example.com/v1"`
		Services  []*url.URL `env:"SERVICES" default:"postgres://localhost:5432/db,redis://localhost:6379"`
		Empty     []url.URL  `env:"EMPTY_URLS"`
	}

	// Don't set any environment variables to test defaults
	cfg, err := Load(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Endpoints) != 2 {
		t.Errorf("expected 2 default endpoints, got %d", len(cfg.Endpoints))
	}
	if cfg.Endpoints[0].Host != "localhost:8080" {
		t.Errorf("expected first endpoint host 'localhost:8080', got %s", cfg.Endpoints[0].Host)
	}
	if cfg.Endpoints[1].Host != "api.example.com" {
		t.Errorf("expected second endpoint host 'api.example.com', got %s", cfg.Endpoints[1].Host)
	}

	if len(cfg.Services) != 2 {
		t.Errorf("expected 2 default services, got %d", len(cfg.Services))
	}
	if cfg.Services[0] == nil || cfg.Services[0].Scheme != "postgres" {
		t.Error("expected first service to be postgres")
	}
	if cfg.Services[1] == nil || cfg.Services[1].Scheme != "redis" {
		t.Error("expected second service to be redis")
	}

	if cfg.Empty != nil && len(cfg.Empty) > 0 {
		t.Error("Empty slice should remain empty when no env var or default is set")
	}
}

func TestUnixSocketURLSupport(t *testing.T) {
	type Config struct {
		DatabaseURL url.URL  `env:"DATABASE_URL"`
		SocketURL   *url.URL `env:"SOCKET_URL"`
	}

	// Test both standard TCP and Unix socket PostgreSQL URLs
	tests := []struct {
		name         string
		envVars      map[string]string
		expectError  bool
		checkResults func(t *testing.T, cfg *Config)
	}{
		{
			name: "Unix socket PostgreSQL URL",
			envVars: map[string]string{
				"DATABASE_URL": "postgresql://user:password@/mydb?host=/var/run/postgresql",
				"SOCKET_URL":   "postgresql://admin:secret@/testdb?host=/tmp/.s.PGSQL.5432",
			},
			expectError: false,
			checkResults: func(t *testing.T, cfg *Config) {
				// Check main database URL
				if cfg.DatabaseURL.Scheme != "postgresql" {
					t.Errorf("expected scheme postgresql, got %s", cfg.DatabaseURL.Scheme)
				}
				if cfg.DatabaseURL.Host != "" {
					t.Errorf("expected empty host for Unix socket, got %s", cfg.DatabaseURL.Host)
				}
				if cfg.DatabaseURL.Path != "/mydb" {
					t.Errorf("expected path /mydb, got %s", cfg.DatabaseURL.Path)
				}

				// Check query parameters for socket path
				params := cfg.DatabaseURL.Query()
				if socketPath := params.Get("host"); socketPath != "/var/run/postgresql" {
					t.Errorf("expected socket path /var/run/postgresql, got %s", socketPath)
				}

				if cfg.DatabaseURL.User.Username() != "user" {
					t.Errorf("expected username user, got %s", cfg.DatabaseURL.User.Username())
				}
				password, _ := cfg.DatabaseURL.User.Password()
				if password != "password" {
					t.Errorf("expected password 'password', got %s", password)
				}

				// Check socket URL pointer
				if cfg.SocketURL == nil {
					t.Error("SocketURL should not be nil")
				} else {
					if cfg.SocketURL.Scheme != "postgresql" {
						t.Errorf("expected scheme postgresql, got %s", cfg.SocketURL.Scheme)
					}
					if cfg.SocketURL.Host != "" {
						t.Errorf("expected empty host for Unix socket, got %s", cfg.SocketURL.Host)
					}
					socketParams := cfg.SocketURL.Query()
					if socketPath := socketParams.Get("host"); socketPath != "/tmp/.s.PGSQL.5432" {
						t.Errorf("expected socket path /tmp/.s.PGSQL.5432, got %s", socketPath)
					}
				}
			},
		},
		{
			name: "Mixed TCP and Unix socket URLs",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/tcpdb",
				"SOCKET_URL":   "postgresql://admin:secret@/socketdb?host=/run/postgresql",
			},
			expectError: false,
			checkResults: func(t *testing.T, cfg *Config) {
				// TCP URL should have host
				if cfg.DatabaseURL.Host != "localhost:5432" {
					t.Errorf("expected TCP host localhost:5432, got %s", cfg.DatabaseURL.Host)
				}

				// Unix socket URL should have empty host but socket in query
				if cfg.SocketURL.Host != "" {
					t.Errorf("expected empty host for Unix socket, got %s", cfg.SocketURL.Host)
				}
				socketParams := cfg.SocketURL.Query()
				if socketPath := socketParams.Get("host"); socketPath != "/run/postgresql" {
					t.Errorf("expected socket path /run/postgresql, got %s", socketPath)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for key := range tt.envVars {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load(Config{})

			if tt.expectError && err == nil {
				t.Error("expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkResults != nil {
				tt.checkResults(t, &cfg)
			}

			// Cleanup
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestUnixSocketURLPasswordMasking(t *testing.T) {
	type Config struct {
		SocketURL   url.URL  `env:"SOCKET_URL"`
		SocketPtr   *url.URL `secret:"SOCKET_SECRET"`
		MixedConfig struct {
			TCP    url.URL  `env:"TCP_URL"`
			Socket *url.URL `secret:"UNIX_SOCKET"`
		}
	}

	os.Setenv("SOCKET_URL", "postgresql://user:socketpass@/mydb?host=/var/run/postgresql")
	os.Setenv("SOCKET_SECRET", "postgresql://admin:topsecret@/secretdb?host=/tmp/.s.PGSQL.5432&sslmode=disable")
	os.Setenv("TCP_URL", "postgres://user:tcppass@localhost:5432/tcpdb")
	os.Setenv("UNIX_SOCKET", "postgresql://root:unixsecret@/unixdb?host=/run/postgresql")

	defer func() {
		os.Unsetenv("SOCKET_URL")
		os.Unsetenv("SOCKET_SECRET")
		os.Unsetenv("TCP_URL")
		os.Unsetenv("UNIX_SOCKET")
	}()
	var cfg Config
	var err error
	if cfg, err = Load(Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prettyStr := PrettyString(cfg)
	t.Logf("Pretty string output:\n%s", prettyStr)

	// Parse the JSON output to check masking
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(prettyStr), &result); err != nil {
		t.Fatalf("failed to parse PrettyString output: %v", err)
	}

	// Check that passwords are masked in Unix socket URLs
	socketURL, ok := result["SOCKET_URL"].(string)
	if !ok {
		t.Error("SOCKET_URL should be a string in output")
	} else if !strings.Contains(socketURL, "user:") || (!strings.Contains(socketURL, ":***@") && !strings.Contains(socketURL, ":%2A%2A%2A@")) {
		t.Errorf("Unix socket URL password should be masked, got: %s", socketURL)
	}

	secretURL, ok := result["SOCKET_SECRET"].(string)
	if !ok {
		t.Error("SOCKET_SECRET should be a string in output")
	} else if secretURL != "***" {
		t.Errorf("Secret Unix socket URL should be completely masked, got: %s", secretURL)
	}

	// Check nested struct
	mixedConfig, ok := result["MixedConfig"].(map[string]interface{})
	if !ok {
		t.Error("MixedConfig should be a map in output")
	} else {
		tcpURL, ok := mixedConfig["TCP_URL"].(string)
		if !ok {
			t.Error("TCP_URL should be a string in nested config")
		} else if !strings.Contains(tcpURL, "user:") || (!strings.Contains(tcpURL, ":***@") && !strings.Contains(tcpURL, ":%2A%2A%2A@")) {
			t.Errorf("TCP URL password should be masked, got: %s", tcpURL)
		}

		unixSocket, ok := mixedConfig["UNIX_SOCKET"].(string)
		if !ok {
			t.Error("UNIX_SOCKET should be a string in nested config")
		} else if unixSocket != "***" {
			t.Errorf("Unix socket secret should be completely masked, got: %s", unixSocket)
		}
	}
}

// Helper functions
func mustParseURL(rawURL string) url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return *u
}

func mustParseURLPtr(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
