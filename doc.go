// Package gonfig provides a small, type-safe configuration loader for Go application//	// Crypto keys (PEM format)
//
//	PrivateKey rsa.PrivateKey `secret:"PRIVATE_KEY"`
//
//	// Expression language for business rules
//	AccessRule *vm.Program `env:"ACCESS_RULE" default:"user.role == 'admin'"`
//	Rules      []*vm.Program `env:"BUSINESS_RULES"`
//
//	// Database URLs with support for both TCP and Unix sockets/
//
// # Features
//
//   - Loads struct-tagged settings directly from environment variables
//   - Optional .env file support
//   - Default values and CSV slices parsing
//   - Extended parsing for time.Duration, uuid.UUID, decimal.Decimal, vm.Program (expr), and other specialized types
//   - Secret masking for sensitive configuration values
//   - Support for nested structs (both value and pointer types)
//   - Comprehensive type support including crypto keys, network addresses, and Kubernetes resource quantities
//
// # Supported Types
//
// The library supports a wide range of Go types:
//   - Basic types: string, bool, int (all sizes), float32, float64
//   - Collections: slices of supported types
//   - Time types: time.Duration, time.Time
//   - Network types: net.IP, mail.Address, url.URL
//   - Crypto types: rsa.PrivateKey, ecdsa.PrivateKey (from PEM format)
//   - Specialized types: uuid.UUID, decimal.Decimal, big.Int, slog.Level
//   - Kubernetes types: resource.Quantity
//   - Expression language: vm.Program (expr-lang/expr for business rules and validation)
//   - Any type implementing encoding.TextUnmarshaler
//   - Nested structs (recursive processing)
//
// # Struct Tags
//
// Configuration is controlled through struct tags:
//   - `env:"VAR_NAME"` - Maps field to environment variable
//   - `secret:"VAR_NAME"` - Maps field to environment variable but masks it in output
//   - `default:"value"` - Provides fallback value when environment variable is not set
//   - `required:"true"` - Makes field required (fails if not set and no default)
//
// # Quick Start
//
//	package main
//
//	import (
//		"fmt"
//		"log"
//
//		"github.com/vivaneiona/gonfig"
//	)
//
//	type Config struct {
//		Port   int      `env:"PORT"   default:"8080"`
//		Host   string   `env:"HOST"   default:"localhost"`
//		Debug  bool     `env:"DEBUG"  default:"false"`
//		APIKey string   `secret:"API_KEY"`
//		Tags   []string `env:"TAGS"   default:"web,api"`
//
//		Database struct {
//			Host string `env:"DB_HOST" default:"localhost"`
//			Port int    `env:"DB_PORT" default:"5432"`
//		}
//
//		DatabaseURL url.URL `env:"DATABASE_URL"`
//	}
//
//	func main() {
//		cfg, err := gonfig.Load(Config{})
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		fmt.Println(gonfig.PrettyString(cfg)) // secrets are masked
//	}
//
// # Advanced Usage
//
// The library supports complex configuration scenarios:
//
//	type AdvancedConfig struct {
//		// Kubernetes resource quantities
//		DefaultMem  resource.Quantity `env:"DEFAULT_MEM" default:"1Gi"`
//		DefaultDisk resource.Quantity `env:"DEFAULT_DISK" default:"10G"`
//
//		// Network configuration
//		IPAddress net.IP        `env:"IP_ADDRESS"`
//		EmailAddr mail.Address  `env:"EMAIL_ADDR"`
//
//		// Logging
//		LogLevel slog.Level `env:"LOG_LEVEL"`
//
//		// Crypto keys (PEM format)
//		PrivateKey rsa.PrivateKey `secret:"PRIVATE_KEY"`
//
//		// Database URLs with support for both TCP and Unix sockets
//		DatabaseURL url.URL `env:"DATABASE_URL" default:"postgres://user:pass@localhost:5432/mydb"`
//		SocketURL   url.URL `env:"SOCKET_URL" default:"postgresql://user:pass@/mydb?host=/var/run/postgresql"`
//	}
//
// # Environment File Support
//
// Load configuration with optional .env file support:
//
//	cfg, err := gonfig.LoadWithDotenv(Config{}, ".env", ".env.local")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # API Reference
//
// The package provides three main functions:
//
//	func Load[T any](cfg T) (T, error)                    // Load from environment variables only
//	func LoadWithDotenv[T any](cfg T, paths ...string) (T, error) // Load with .env file support
//	func PrettyString(v any) string                       // Format config with masked secrets
//
// # Error Handling
//
// The library provides detailed error messages for configuration issues:
//   - Missing required fields
//   - Type conversion failures
//   - Invalid default values
//   - Malformed PEM keys or other specialized formats
//
// All errors include context about the field name and expected format to aid in debugging.
package gonfig
