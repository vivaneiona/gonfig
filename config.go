// Package gonfig provides type-safe configuration loading from environment variables
// with support for defaults, secret masking, nested structs, and .env files.
//
// The library uses struct tags to control configuration loading:
//   - `env` tag: Maps struct fields to environment variables
//   - `secret` tag: Maps struct fields to environment variables but masks them in output
//   - `default` tag: Provides fallback values when environment variables are not set
//   - `required` tag: Makes fields required (fails if not set and no default)
//
// Supported types: string, bool, int (all sizes), float32, float64, slices, nested structs,
// time.Duration, time.Time, slog.Level, big.Int, decimal.Decimal, url.URL, net.IP, mail.Address,
// uuid.UUID, resource.Quantity, rsa.PrivateKey, ecdsa.PrivateKey (from PEM), vm.Program (expr-lang/expr),
// and any type implementing encoding.TextUnmarshaler
//
// New: Nested structs (value or pointer) are fully supported with recursive processing.
//
// Example usage:
//
//	type Config struct {
//	    Port   int    `env:"PORT" default:"8080"`
//	    APIKey string `secret:"API_KEY" required:"true"`
//	    DB     struct {
//	        Host string `env:"DB_HOST" default:"localhost"`
//	        Port int    `env:"DB_PORT" default:"5432"`
//	    }
//	    // PostgreSQL URLs - supports both TCP and Unix socket formats
//	    DatabaseURL url.URL `env:"DATABASE_URL" default:"postgres://user:pass@localhost:5432/mydb"`
//	    SocketURL   url.URL `env:"SOCKET_URL" default:"postgresql://user:pass@/mydb?host=/var/run/postgresql"`
//	}
//
//	cfg, err := gonfig.Load(Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Info("config loaded", "cfg", gonfig.PrettyString(cfg))
package gonfig

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/mail"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"k8s.io/apimachinery/pkg/api/resource"
)

// mask returns a masked version of the secret string.
// It keeps the first 3 characters visible and replaces the rest with asterisks.
// For strings with 3 or fewer characters, all characters are replaced with asterisks.
//
// Examples:
//   - mask("") returns ""
//   - mask("a") returns "*"
//   - mask("abc") returns "***"
//   - mask("secret123") returns "sec*****"
func mask(secret string) string {
	const keep = 3
	n := len(secret)
	if n <= keep {
		return strings.Repeat("*", n)
	}
	return secret[:keep] + strings.Repeat("*", n-keep)
}

// parseWithRegistry checks for explicit parsers first, then factories, before falling back to parseScalar.
func parseWithRegistry(raw string, t reflect.Type, kind reflect.Kind, bits int) (any, error) {
	// Check explicit registered parsers first (highest priority)
	if fn, ok := customParsers[t]; ok {
		return fn(raw)
	}

	// Check parser factories (in registration order)
	for _, factory := range parserFactories {
		if parser := factory(t); parser != nil {
			return parser(raw)
		}
	}

	// fallback to existing parseScalar logic
	return parseScalar(raw, kind, bits)
}

// parseScalar parses a string value into the appropriate type based on reflect.Kind
func parseScalar(raw string, kind reflect.Kind, bits int) (any, error) {
	switch kind {
	case reflect.String:
		return raw, nil
	case reflect.Bool:
		return strconv.ParseBool(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special case: duration parsing for int64 fields ending with 's'
		if kind == reflect.Int64 && strings.HasSuffix(raw, "s") {
			d, err := time.ParseDuration(raw)
			return int64(d), err
		}
		return strconv.ParseInt(raw, 10, bits)
	case reflect.Float32, reflect.Float64:
		return strconv.ParseFloat(raw, bits)
	default:
		return nil, fmt.Errorf("unsupported scalar kind %s", kind)
	}
}

// PrettyString returns a JSON-formatted string representation of the configuration
// with secret fields automatically masked for safe logging and debugging.
// Now supports nested structs (value or pointer) with recursive processing.
//
// Fields tagged with `secret` will have their values masked using the mask function.
// The function uses struct tags to determine field names in the output:
//   - `env` tag value is used as the key name
//   - `secret` tag value is used as the key name (and value is masked)
//   - If no tag is present, the struct field name is used
//
// Example:
//
//	type Config struct {
//	    Port   int    `env:"PORT"`
//	    APIKey string `secret:"API_KEY"`
//	    DB     struct {
//	        Host string `env:"DB_HOST"`
//	    }
//	//	cfg := &Config{Port: 8080, APIKey: "secret123"}
//	fmt.Println(PrettyString(cfg))
//	// Output: {"PORT": 8080, "API_KEY": "sec*****", "DB": {"DB_HOST": ""}}
func PrettyString(c any) string {
	rv := reflect.ValueOf(c)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Sprintf("%T is not a struct", c)
	}

	obj := buildSafeMap(rv)
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("error pretty-printing config: %v", err)
	}
	return string(b)
}

// isCustomParsedType checks if a type has a custom parser registered or can be handled by a factory
func isCustomParsedType(t reflect.Type) bool {
	// Check explicit parsers first
	if _, exists := customParsers[t]; exists {
		return true
	}

	// For structs, only consider them custom parsed if they explicitly implement TextUnmarshaler
	// and are meant to be parsed from strings (like time.Time, url.URL, etc.)
	if t.Kind() == reflect.Struct {
		// Check if this struct actually implements TextUnmarshaler
		textUnmarshalerType := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
		if reflect.PointerTo(t).Implements(textUnmarshalerType) {
			return true
		}
		return false
	}

	// For non-struct types, check if any factory can handle this type
	for _, factory := range parserFactories {
		if parser := factory(t); parser != nil {
			return true
		}
	}

	return false
}

// isURLType checks if the type is url.URL or *url.URL
func isURLType(t reflect.Type) bool {
	if t == reflect.TypeOf(url.URL{}) {
		return true
	}
	if t == reflect.TypeOf(&url.URL{}) {
		return true
	}
	return false
}

// maskURLPassword masks the password in a URL for safe logging
func maskURLPassword(val any) any {
	switch u := val.(type) {
	case url.URL:
		if u.User != nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				// Create a copy and mask the password
				masked := u
				masked.User = url.UserPassword(u.User.Username(), "***")
				return masked.String()
			}
		}
		return u.String()
	case *url.URL:
		if u != nil && u.User != nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				// Create a copy and mask the password
				masked := *u
				masked.User = url.UserPassword(u.User.Username(), "***")
				return masked.String()
			}
			return u.String()
		}
		if u == nil {
			return nil
		}
		return u.String()
	default:
		return val
	}
}

// buildSafeMap recursively builds a safe map representation of a struct
// with secret fields masked and nested structs preserved
func buildSafeMap(val reflect.Value) map[string]any {
	typ := val.Type()
	out := make(map[string]any, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		fv := val.Field(i)

		// Skip unexported fields
		if !fv.CanInterface() {
			continue
		}

		// use env tag or secret tag as key, fallback to field name
		key := sf.Tag.Get("env")
		if key == "" {
			key = sf.Tag.Get("secret")
		}
		if key == "" {
			key = sf.Name
		}

		switch {
		case sf.Tag.Get("secret") != "":
			// mask secret fields
			if fv.Kind() == reflect.Slice {
				// Handle secret slices by masking each element
				slice := make([]interface{}, fv.Len())
				for i := 0; i < fv.Len(); i++ {
					elem := fv.Index(i)
					if s, ok := elem.Interface().(string); ok {
						slice[i] = mask(s)
					} else {
						slice[i] = "***"
					}
				}
				out[key] = slice
			} else if s, ok := fv.Interface().(string); ok {
				out[key] = mask(s)
			} else {
				out[key] = "***"
			}
		case isURLType(fv.Type()):
			// Handle special types like url.URL
			out[key] = maskURLPassword(fv.Interface())
		case fv.Kind() == reflect.Slice:
			// Handle regular slices
			slice := make([]interface{}, fv.Len())
			for i := 0; i < fv.Len(); i++ {
				elem := fv.Index(i)
				elemInterface := elem.Interface()

				// Check if slice element is a URL type
				if isURLType(elem.Type()) {
					slice[i] = maskURLPassword(elemInterface)
				} else {
					slice[i] = elemInterface
				}
			}
			out[key] = slice
		case fv.Kind() == reflect.Struct:
			// recursively handle nested structs
			out[key] = buildSafeMap(fv)
		case fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Struct:
			// recursively handle pointer to structs
			if fv.IsNil() {
				out[key] = nil
			} else {
				out[key] = buildSafeMap(fv.Elem())
			}
		default:
			// regular fields
			out[key] = fv.Interface()
		}
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make(map[string]any, len(keys))
	for _, k := range keys {
		sorted[k] = out[k]
	}
	return sorted
}

// Load populates a configuration struct from environment variables using reflection.
// Now supports nested structs (value or pointer) with recursive processing.
// It supports the following struct tags:
//   - `env:"ENV_VAR"`: Maps the field to the specified environment variable
//   - `secret:"SECRET_VAR"`: Maps the field to the specified environment variable (for secrets)
//   - `default:"value"`: Sets a default value if the environment variable is not set
//   - `required:"true"`: Makes the field required (fails if not set and no default)
//
// Supported field types:
//   - string
//   - bool (parsed using strconv.ParseBool)
//   - int, int8, int16, int32, int64 (parsed using strconv.ParseInt)
//   - float32, float64 (parsed using strconv.ParseFloat)
//   - slices of the above types (comma-separated values)
//   - nested structs (value or pointer)
//   - time.Duration for int64 fields (when value ends with 's', 'ms', etc.)
//   - time.Duration (parsed using time.ParseDuration)
//   - time.Time (RFC3339 format or Unix seconds)
//   - log/slog.Level (debug|info|warn|error or integer)
//   - math/big.Int (base-10 integer strings)
//   - github.com/shopspring/decimal.Decimal (exact decimal arithmetic)
//   - url.URL (parsed using url.Parse, supports TCP and Unix socket PostgreSQL URLs)
//     Examples: postgres://user:pass@host:port/db, postgresql://user:pass@/db?host=/socket/path
//   - net.IP (IPv4 and IPv6 addresses)
//   - net/mail.Address (email addresses with optional display names)
//   - github.com/google/uuid.UUID (UUID strings)
//   - k8s.io/apimachinery/pkg/api/resource.Quantity (Kubernetes resource units like 250m, 1.5Gi)
//   - crypto/rsa.PrivateKey (RSA private keys from PEM format)
//   - crypto/ecdsa.PrivateKey (ECDSA private keys from PEM format)
//   - github.com/expr-lang/expr/vm.Program (compiled expressions for business rules and validation)
//   - Any type implementing encoding.TextUnmarshaler
//
// The function returns an error if:
//   - An unsupported field type is encountered
//   - A required field is missing
//   - Type conversion fails
//
// Example:
//
//	type Config struct {
//	    Port   int    `env:"PORT" default:"8080"`
//	    Debug  bool   `env:"DEBUG" default:"false"`
//	    APIKey string `secret:"API_KEY" required:"true"`
//	    DB     struct {
//	        Host string `env:"DB_HOST" default:"localhost"`
//	    }
//	}
//
//	cfg, err := Load(Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
func Load[T any](config T) (T, error) {
	rv := reflect.ValueOf(config)

	// Handle the case where config is already a pointer to a struct
	if rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Struct {
		err := loadStruct(rv.Elem())
		return config, err
	}

	// Handle the case where config is a struct value
	if rv.Kind() == reflect.Struct {
		// Create a pointer to the struct for modification
		cfg := &config
		rv := reflect.ValueOf(cfg)
		err := loadStruct(rv.Elem())
		return config, err
	}

	var zero T
	return zero, fmt.Errorf("config must be struct or pointer to struct, got %T", config)
}

// loadStruct recursively loads configuration into a struct value
func loadStruct(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		fv := val.Field(i)

		// Skip unexported fields
		if !fv.CanSet() {
			continue
		}

		// Handle nested structs recursively (but not custom parsed types)
		if fv.Kind() == reflect.Struct && !isCustomParsedType(fv.Type()) {
			if err := loadStruct(fv); err != nil {
				return err
			}
			continue
		}
		if fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Struct && !isCustomParsedType(fv.Type()) {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			if err := loadStruct(fv.Elem()); err != nil {
				return err
			}
			continue
		}

		// determine key (env or secret tag)
		key := sf.Tag.Get("env")
		if key == "" {
			key = sf.Tag.Get("secret")
		}
		if key == "" {
			key = sf.Name
		}

		// pick up env or fallback to default tag (only if field is zero value)
		raw, ok := os.LookupEnv(key)
		if !ok {
			// Only use default if the field currently has a zero value
			if fv.IsZero() {
				raw = sf.Tag.Get("default")
			} else {
				// Field already has a non-zero value, skip setting it
				continue
			}
		}
		if raw == "" && sf.Tag.Get("required") == "true" {
			return fmt.Errorf("required env %q missing", key)
		}
		if raw == "" { // nothing to set
			continue
		}

		// Handle slices (but not if the slice type itself has a custom parser like net.IP)
		if fv.Kind() == reflect.Slice && !isCustomParsedType(fv.Type()) {
			elemType := fv.Type().Elem()
			elemKind := elemType.Kind()
			slice := reflect.MakeSlice(fv.Type(), 0, 0)

			// Handle empty string case - create empty slice
			if raw == "" {
				fv.Set(slice)
				continue
			}

			for _, part := range strings.Split(raw, ",") {
				part = strings.TrimSpace(part)
				// Skip empty parts
				if part == "" {
					continue
				}

				parsed, err := parseWithRegistry(part, elemType, elemKind, getBits(elemType))
				if err != nil {
					return fmt.Errorf("field %s: %w", sf.Name, err)
				}

				// Special handling for custom parsers in slices
				if _, isCustom := customParsers[elemType]; isCustom {
					slice = reflect.Append(slice, reflect.ValueOf(parsed))
				} else {
					slice = reflect.Append(slice, reflect.ValueOf(parsed).Convert(elemType))
				}
			}
			fv.Set(slice)
			continue
		}

		// Handle scalar types
		parsed, err := parseWithRegistry(raw, fv.Type(), fv.Kind(), getBits(fv.Type()))
		if err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}

		// Special handling for custom parsers
		if _, isCustom := customParsers[fv.Type()]; isCustom {
			fv.Set(reflect.ValueOf(parsed))
		} else {
			fv.Set(reflect.ValueOf(parsed).Convert(fv.Type()))
		}
	}

	return nil
}

// getBits safely returns the bit size for numeric types, 0 for others
func getBits(t reflect.Type) int {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64:
		return t.Bits()
	default:
		return 0
	}
}

// LoadWithDotenv loads configuration from environment variables with support for .env files.
// It first attempts to load a .env file using godotenv, then calls Load to populate
// the configuration struct from environment variables.
//
// The function loads environment variables in this precedence order:
//  1. Existing environment variables (highest priority)
//  2. Variables from .env file
//  3. Default values from struct tags (lowest priority)
//
// If the .env file doesn't exist or can't be loaded, the error is silently ignored
// and the function continues with existing environment variables.
//
// Parameters:
//   - config: Pointer to a configuration struct with tagged fields
//   - dotenvPath: Optional path to .env file (defaults to ".env" in current directory)
//
// Example:
//
//	type Config struct {
//	    Port   int    `env:"PORT" default:"8080"`
//	    APIKey string `secret:"API_KEY"`
//	}
//
//	cfg := &Config{}
//	loaded := LoadWithDotenv(cfg, "config/.env")
//
// LoadWithDotenv loads a .env file first, then calls Load to populate the configuration.
// It accepts an optional path to the .env file; if not provided, it defaults to ".env".
// Returns the populated configuration struct and any error encountered.
//
// Example:
//
//	cfg, err := LoadWithDotenv(Config{}, ".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadWithDotenv[T any](config T, dotenvPath ...string) (T, error) {
	// Load .env file if specified, otherwise try to load from current directory
	var envPath string
	if len(dotenvPath) > 0 {
		envPath = dotenvPath[0]
	} else {
		envPath = ".env"
	}

	// Load .env file, ignore error if file doesn't exist
	_ = godotenv.Load(envPath)

	// Use the regular Load function after loading .env
	return Load(config)
}

// parserFunc takes the raw string and returns the parsed value or an error.
type parserFunc func(raw string) (any, error)

// parserFactory generates a parser function for a given type, or returns nil if not supported.
type parserFactory func(t reflect.Type) parserFunc

// registry of custom parsers
var customParsers = make(map[reflect.Type]parserFunc)

// registry of parser factories (checked in order)
var parserFactories []parserFactory

// RegisterParser lets users plug in custom type parsers.
// Call this in your init() or main() before Load.
func RegisterParser(typ reflect.Type, fn parserFunc) {
	customParsers[typ] = fn
}

// RegisterParserFactory lets users plug in factory functions that can generate
// parsers for entire categories of types (e.g., anything implementing TextUnmarshaler).
// Factories are checked in registration order before falling back to explicit parsers.
func RegisterParserFactory(factory parserFactory) {
	parserFactories = append(parserFactories, factory)
}

func init() {
	// Register the TextUnmarshaler factory first - this unlocks dozens of std-lib and third-party types
	RegisterParserFactory(func(t reflect.Type) parserFunc {
		// Check if the type implements encoding.TextUnmarshaler
		textUnmarshalerType := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

		// Handle pointer types
		targetType := t
		if t.Kind() == reflect.Pointer {
			targetType = t.Elem()
		}

		if reflect.PointerTo(targetType).Implements(textUnmarshalerType) {
			return func(raw string) (any, error) {
				// Create a new instance of the target type
				v := reflect.New(targetType).Interface().(encoding.TextUnmarshaler)
				if err := v.UnmarshalText([]byte(raw)); err != nil {
					return nil, fmt.Errorf("failed to unmarshal text: %w", err)
				}

				// Return the appropriate type (value or pointer)
				if t.Kind() == reflect.Pointer {
					return v, nil
				}
				return reflect.ValueOf(v).Elem().Interface(), nil
			}
		}
		return nil
	})

	// Register built-in url.URL parsers (explicit for better performance)
	RegisterParser(reflect.TypeOf(url.URL{}), func(raw string) (any, error) {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", raw, err)
		}
		return *u, nil
	})

	RegisterParser(reflect.TypeOf(&url.URL{}), func(raw string) (any, error) {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", raw, err)
		}
		return u, nil
	})

	// Register time.Duration parser (explicit for better performance than TextUnmarshaler)
	RegisterParser(reflect.TypeOf(time.Duration(0)), func(raw string) (any, error) {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", raw, err)
		}
		return d, nil
	})

	// Register time.Time parsers (RFC3339 and Unix seconds)
	RegisterParser(reflect.TypeOf(time.Time{}), func(raw string) (any, error) {
		// Try RFC3339 format first
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t, nil
		}

		// Try Unix seconds as fallback
		if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return time.Unix(unix, 0), nil
		}

		return nil, fmt.Errorf("invalid time %q: must be RFC3339 format or Unix seconds", raw)
	})

	// Register *time.Time parser
	RegisterParser(reflect.TypeOf(&time.Time{}), func(raw string) (any, error) {
		// Try RFC3339 format first
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return &t, nil
		}

		// Try Unix seconds as fallback
		if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
			t := time.Unix(unix, 0)
			return &t, nil
		}

		return nil, fmt.Errorf("invalid time %q: must be RFC3339 format or Unix seconds", raw)
	})

	// Register slog.Level parser
	RegisterParser(reflect.TypeOf(slog.Level(0)), func(raw string) (any, error) {
		switch strings.ToLower(raw) {
		case "debug":
			return slog.LevelDebug, nil
		case "info":
			return slog.LevelInfo, nil
		case "warn", "warning":
			return slog.LevelWarn, nil
		case "error":
			return slog.LevelError, nil
		default:
			// Try parsing as integer level
			if level, err := strconv.Atoi(raw); err == nil {
				return slog.Level(level), nil
			}
			return nil, fmt.Errorf("invalid slog level %q: must be debug|info|warn|error or integer", raw)
		}
	})

	// Register *slog.Level parser
	RegisterParser(reflect.TypeOf((*slog.Level)(nil)), func(raw string) (any, error) {
		switch strings.ToLower(raw) {
		case "debug":
			level := slog.LevelDebug
			return &level, nil
		case "info":
			level := slog.LevelInfo
			return &level, nil
		case "warn", "warning":
			level := slog.LevelWarn
			return &level, nil
		case "error":
			level := slog.LevelError
			return &level, nil
		default:
			// Try parsing as integer level
			if levelInt, err := strconv.Atoi(raw); err == nil {
				level := slog.Level(levelInt)
				return &level, nil
			}
			return nil, fmt.Errorf("invalid slog level %q: must be debug|info|warn|error or integer", raw)
		}
	})

	// Register big.Int parsers (explicit since big.Int doesn't implement TextUnmarshaler in the way we want)
	RegisterParser(reflect.TypeOf(&big.Int{}), func(raw string) (any, error) {
		bi := new(big.Int)
		if _, ok := bi.SetString(raw, 10); !ok {
			return nil, fmt.Errorf("invalid big.Int %q: must be base-10 integer", raw)
		}
		return bi, nil
	})

	RegisterParser(reflect.TypeOf(big.Int{}), func(raw string) (any, error) {
		bi := new(big.Int)
		if _, ok := bi.SetString(raw, 10); !ok {
			return nil, fmt.Errorf("invalid big.Int %q: must be base-10 integer", raw)
		}
		return *bi, nil
	})

	// Register decimal.Decimal parsers
	RegisterParser(reflect.TypeOf(decimal.Decimal{}), func(raw string) (any, error) {
		d, err := decimal.NewFromString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal %q: %w", raw, err)
		}
		return d, nil
	})

	RegisterParser(reflect.TypeOf(&decimal.Decimal{}), func(raw string) (any, error) {
		d, err := decimal.NewFromString(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid decimal %q: %w", raw, err)
		}
		return &d, nil
	})

	// Register net.IP parsers (net.IP is []byte, so needs explicit handling)
	RegisterParser(reflect.TypeOf(net.IP{}), func(raw string) (any, error) {
		ip := net.ParseIP(raw)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address %q", raw)
		}
		return ip, nil
	})

	RegisterParser(reflect.TypeOf(&net.IP{}), func(raw string) (any, error) {
		ip := net.ParseIP(raw)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address %q", raw)
		}
		return &ip, nil
	})

	// Register mail.Address parsers (special parsing logic needed)
	RegisterParser(reflect.TypeOf(mail.Address{}), func(raw string) (any, error) {
		addr, err := mail.ParseAddress(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid email address %q: %w", raw, err)
		}
		return *addr, nil
	})

	RegisterParser(reflect.TypeOf(&mail.Address{}), func(raw string) (any, error) {
		addr, err := mail.ParseAddress(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid email address %q: %w", raw, err)
		}
		return addr, nil
	})

	// Register Kubernetes resource.Quantity parsers (cloud-native resource units)
	RegisterParser(reflect.TypeOf(resource.Quantity{}), func(raw string) (any, error) {
		q, err := resource.ParseQuantity(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid k8s quantity %q: %w", raw, err)
		}
		return q, nil
	})

	RegisterParser(reflect.TypeOf(&resource.Quantity{}), func(raw string) (any, error) {
		q, err := resource.ParseQuantity(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid k8s quantity %q: %w", raw, err)
		}
		return &q, nil
	})

	// Register RSA private key parsers (for JWT signers from PEM in K8s secrets)
	RegisterParser(reflect.TypeOf(&rsa.PrivateKey{}), func(raw string) (any, error) {
		block, _ := pem.Decode([]byte(raw))
		if block == nil {
			return nil, fmt.Errorf("invalid PEM format for RSA private key")
		}

		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
			}
			return key, nil
		case "PRIVATE KEY":
			// Try PKCS#8 format
			keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
			}
			if rsaKey, ok := keyInterface.(*rsa.PrivateKey); ok {
				return rsaKey, nil
			}
			return nil, fmt.Errorf("PKCS#8 key is not an RSA private key")
		default:
			return nil, fmt.Errorf("unsupported PEM block type for RSA private key: %s", block.Type)
		}
	})

	RegisterParser(reflect.TypeOf(rsa.PrivateKey{}), func(raw string) (any, error) {
		block, _ := pem.Decode([]byte(raw))
		if block == nil {
			return nil, fmt.Errorf("invalid PEM format for RSA private key")
		}

		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
			}
			return *key, nil
		case "PRIVATE KEY":
			// Try PKCS#8 format
			keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
			}
			if rsaKey, ok := keyInterface.(*rsa.PrivateKey); ok {
				return *rsaKey, nil
			}
			return nil, fmt.Errorf("PKCS#8 key is not an RSA private key")
		default:
			return nil, fmt.Errorf("unsupported PEM block type for RSA private key: %s", block.Type)
		}
	})

	// Register ECDSA private key parsers (for JWT signers from PEM in K8s secrets)
	RegisterParser(reflect.TypeOf(&ecdsa.PrivateKey{}), func(raw string) (any, error) {
		block, _ := pem.Decode([]byte(raw))
		if block == nil {
			return nil, fmt.Errorf("invalid PEM format for ECDSA private key")
		}

		switch block.Type {
		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse EC private key: %w", err)
			}
			return key, nil
		case "PRIVATE KEY":
			// Try PKCS#8 format
			keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
			}
			if ecdsaKey, ok := keyInterface.(*ecdsa.PrivateKey); ok {
				return ecdsaKey, nil
			}
			return nil, fmt.Errorf("PKCS#8 key is not an ECDSA private key")
		default:
			return nil, fmt.Errorf("unsupported PEM block type for ECDSA private key: %s", block.Type)
		}
	})

	RegisterParser(reflect.TypeOf(ecdsa.PrivateKey{}), func(raw string) (any, error) {
		block, _ := pem.Decode([]byte(raw))
		if block == nil {
			return nil, fmt.Errorf("invalid PEM format for ECDSA private key")
		}

		switch block.Type {
		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse EC private key: %w", err)
			}
			return *key, nil
		case "PRIVATE KEY":
			// Try PKCS#8 format
			keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
			}
			if ecdsaKey, ok := keyInterface.(*ecdsa.PrivateKey); ok {
				return *ecdsaKey, nil
			}
			return nil, fmt.Errorf("PKCS#8 key is not an ECDSA private key")
		default:
			return nil, fmt.Errorf("unsupported PEM block type for ECDSA private key: %s", block.Type)
		}
	})

	// Register vm.Program parsers (expr-lang/expr expression language)
	RegisterParser(reflect.TypeOf(&vm.Program{}), func(raw string) (any, error) {
		program, err := expr.Compile(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression %q: %w", raw, err)
		}
		return program, nil
	})

	// ✂️  Removed a generic “zero-value struct” parser factory.
	// It interfered with nested *struct initialisation, causing
	// TestNestedPointerStruct to leave pointers nil.
}

// FieldSetting represents metadata about a configuration field
type FieldSetting struct {
	Path      string            // Dot-separated field path (e.g., "DB.Host")
	FieldName string            // Struct field name
	EnvVar    string            // Environment variable name
	Type      string            // Go type name
	Default   string            // Default value from tag
	Required  bool              // Whether field is required
	Secret    bool              // Whether field is marked as secret
	Tags      map[string]string // All struct tags
}

// Settings returns metadata about all configuration fields in the struct.
// It recursively traverses nested structs and collects tag information.
func Settings(config any) []FieldSetting {
	rv := reflect.ValueOf(config)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	var settings []FieldSetting
	collectSettings(rv, "", &settings)
	return settings
}

// collectSettings recursively walks struct fields and collects metadata
func collectSettings(val reflect.Value, prefix string, settings *[]FieldSetting) {
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		fv := val.Field(i)

		// Skip unexported fields
		if !fv.CanInterface() {
			continue
		}

		// Build field path
		fieldPath := sf.Name
		if prefix != "" {
			fieldPath = prefix + "." + sf.Name
		}

		// Handle nested structs recursively (but not custom parsed types)
		// We want to traverse into regular structs, but not into types that have custom parsers
		if fv.Kind() == reflect.Struct {
			// Check if this is a custom parsed type (like time.Time, url.URL, etc.)
			if !isCustomParsedType(fv.Type()) {
				collectSettings(fv, fieldPath, settings)
				continue
			}
		}
		if fv.Kind() == reflect.Pointer && fv.Type().Elem().Kind() == reflect.Struct {
			// For pointer to struct, check if the underlying struct is custom parsed
			if !isCustomParsedType(fv.Type().Elem()) {
				// For pointer to struct, create zero value to traverse
				if fv.IsNil() {
					fv = reflect.New(fv.Type().Elem()).Elem()
				} else {
					fv = fv.Elem()
				}
				collectSettings(fv, fieldPath, settings)
				continue
			}
		}

		// Collect tag metadata
		tags := make(map[string]string)
		tag := sf.Tag

		// Parse common tags
		envVar := tag.Get("env")
		secretVar := tag.Get("secret")
		defaultVal := tag.Get("default")
		requiredVal := tag.Get("required")

		// Store all tags for completeness
		for _, tagName := range []string{"env", "secret", "default", "required", "json", "yaml"} {
			if val := tag.Get(tagName); val != "" {
				tags[tagName] = val
			}
		}

		// Determine environment variable name
		if envVar == "" {
			envVar = secretVar
		}
		if envVar == "" {
			envVar = sf.Name
		}

		// Determine type name
		typeName := fv.Type().String()
		if fv.Kind() == reflect.Slice {
			typeName = "[]" + fv.Type().Elem().String()
		}

		setting := FieldSetting{
			Path:      fieldPath,
			FieldName: sf.Name,
			EnvVar:    envVar,
			Type:      typeName,
			Default:   defaultVal,
			Required:  strings.ToLower(requiredVal) == "true",
			Secret:    secretVar != "",
			Tags:      tags,
		}

		*settings = append(*settings, setting)
	}
}

// FilterSettings returns settings matching the given predicate function
func FilterSettings(settings []FieldSetting, predicate func(FieldSetting) bool) []FieldSetting {
	var filtered []FieldSetting
	for _, setting := range settings {
		if predicate(setting) {
			filtered = append(filtered, setting)
		}
	}
	return filtered
}

// SecretFields returns all fields marked as secrets
func SecretFields(config any) []FieldSetting {
	return FilterSettings(Settings(config), func(s FieldSetting) bool {
		return s.Secret
	})
}

// RequiredFields returns all required fields
func RequiredFields(config any) []FieldSetting {
	return FilterSettings(Settings(config), func(s FieldSetting) bool {
		return s.Required
	})
}
