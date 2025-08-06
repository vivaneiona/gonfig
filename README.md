# ⚙︎ vivaneiona/gonfig

[![Go Reference](https://pkg.go.dev/badge/github.com/vivaneiona/gonfig.svg)](https://pkg.go.dev/github.com/vivaneiona/gonfig)

*A basic, heavy, ad-hoc configuration loader for Go.*

---

## Features

- Loads struct-tagged settings directly from environment variables
- Optional `.env` support
- Defaults and CSV slices
- Extended parsing for `time.Duration`, `uuid.UUID`, `decimal.Decimal`, `vm.Program` (expr), and other specialized types 

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/vivaneiona/gonfig"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)


type Config struct {
	Port int    `env:"PORT"   default:"8080"`
	Host string `env:"HOST"   default:"localhost"`

	Debug  bool     `env:"DEBUG"  default:"false"`
	APIKey string   `secret:"API_KEY"`
	Tags   []string `env:"TAGS"   default:"web,api"`

	DefaultMem  resource.Quantity `env:"DEFAULT_MEM" default:"1Gi"`
	DefaultDisk resource.Quantity `env:"DEFAULT_DISK" default:"10G"`
	IPAddress   net.IP            `env:"IP_ADDRESS"`
	EmailAddr   mail.Address      `env:"EMAIL_ADDR"`

	LogLevel slog.Level `env:"LOG_LEVEL"`

	Database struct {
		Host string `env:"DB_HOST" default:"localhost"`
		Port int    `env:"DB_PORT" default:"5432"`
	}

	DatabaseURL url.URL `env:"DB_URL"`
}

func main() {
	cfg, err := gonfig.Load(Config{})
	if err != nil {
		panic(err)
	}

	fmt.Println(gonfig.PrettyString(cfg)) // secrets masked

	// Use the compiled expression
	if cfg.AccessRule != nil {
		env := map[string]interface{}{
			"user": map[string]interface{}{
				"role":     "user",
				"verified": true,
			},
		}
		result, err := expr.Run(cfg.AccessRule, env)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Access granted: %v\n", result)
	}
}
```

## Custom Types

You can easily add support for your own types by implementing the `encoding.TextUnmarshaler` interface:

```go
package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vivaneiona/gonfig"
)

// Custom type that implements encoding.TextUnmarshaler
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// UnmarshalText implements encoding.TextUnmarshaler
func (l *LogLevel) UnmarshalText(text []byte) error {
	level := LogLevel(strings.ToLower(string(text)))
	switch level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		*l = level
		return nil
	default:
		return errors.New("invalid log level: must be debug, info, warn, or error")
	}
}

// Another custom type for demonstration
type DatabaseDriver string

const (
	PostgreSQL DatabaseDriver = "postgres"
	MySQL     DatabaseDriver = "mysql"
	SQLite    DatabaseDriver = "sqlite"
)

func (d *DatabaseDriver) UnmarshalText(text []byte) error {
	driver := DatabaseDriver(strings.ToLower(string(text)))
	switch driver {
	case PostgreSQL, MySQL, SQLite:
		*d = driver
		return nil
	default:
		return fmt.Errorf("unsupported database driver: %s", text)
	}
}

type Config struct {
	// Your custom types work seamlessly
	LogLevel LogLevel        `env:"LOG_LEVEL" default:"info"`
	Driver   DatabaseDriver  `env:"DB_DRIVER" default:"postgres"`
	
	// Custom types work in slices too
	SupportedDrivers []DatabaseDriver `env:"SUPPORTED_DRIVERS" default:"postgres,mysql"`
	
	// And as pointers
	OptionalLevel *LogLevel `env:"OPTIONAL_LEVEL"`
}

func main() {
	cfg, err := gonfig.Load(Config{})
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("Database Driver: %s\n", cfg.Driver)
	fmt.Printf("Supported Drivers: %v\n", cfg.SupportedDrivers)
}
```

The library automatically detects types that implement `encoding.TextUnmarshaler` and uses them for parsing. This works for:
- Value types and pointer types
- Slices of custom types
- Nested structs containing custom types

## Expression Language Support

gonfig includes built-in support for [expr-lang/expr](https://github.com/expr-lang/expr), a powerful expression language for Go. This allows you to configure business rules, validation logic, and filters using expressions that are compiled once and executed efficiently.

```go
package main

import (
	"fmt"
	"log"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/vivaneiona/gonfig"
)

type Config struct {
	// Business rules
	AccessRule    *vm.Program `env:"ACCESS_RULE" default:"user.role in ['admin', 'moderator']"`
	RateLimit     *vm.Program `env:"RATE_LIMIT" default:"user.requests_per_hour < 1000"`
	FeatureFlags  *vm.Program `env:"FEATURE_FLAGS"`
	
	// Multiple validation rules
	ValidationRules []*vm.Program `env:"VALIDATION_RULES"`
}

func main() {
	cfg, err := gonfig.Load(Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Use compiled expressions for fast evaluation
	user := map[string]interface{}{
		"role":               "admin",
		"requests_per_hour":  500,
		"verified":           true,
	}

	// Check access
	if cfg.AccessRule != nil {
		hasAccess, err := expr.Run(cfg.AccessRule, map[string]interface{}{"user": user})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Access granted: %v\n", hasAccess)
	}

	// Check rate limit
	if cfg.RateLimit != nil {
		withinLimit, err := expr.Run(cfg.RateLimit, map[string]interface{}{"user": user})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Within rate limit: %v\n", withinLimit)
	}
}
```

### Expression Examples

Set expressions via environment variables:

```bash
export ACCESS_RULE="user.role == 'admin' && user.verified == true"
export RATE_LIMIT="user.requests_per_hour < 1000 && user.tier != 'free'"
export FEATURE_FLAGS="user.beta_features == true || user.role in ['admin', 'developer']"
export VALIDATION_RULES="age >= 18,email matches '^[\\w\\.]+@[\\w\\.]+$',country in ['US', 'CA', 'UK']"
```

Benefits:
- **Performance**: Expressions are compiled once at startup, not evaluated repeatedly
- **Safety**: Expression syntax is validated at configuration load time
- **Flexibility**: Support for complex logic including arrays, objects, and built-in functions
- **Type Safety**: Full integration with gonfig's type system and error handling

## API

```go
func Load[T any](cfg *T) *T
func LoadWithDotenv[T any](cfg *T, paths ...string) *T
func PrettyString(v any) string
```

## License

This project is licensed under the MIT License.