# ⚙︎ vivaneiona/gonfig

[![Go Reference](https://pkg.go.dev/badge/github.com/vivaneiona/gonfig.svg)](https://pkg.go.dev/github.com/vivaneiona/gonfig)

*A basic, type-safe configuration loader for Go.*


---

## Features

- Loads struct-tagged settings directly from environment variables
- Optional `.env` support
- Defaults and CSV slices
- Extended parsing for `time.Duration`, `uuid.UUID`, `decimal.Decimal`, and other 

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/vivaneiona/gonfig"
)

type Config struct {
	Port   int      `env:"PORT"   default:"8080"`
	Host   string   `env:"HOST"   default:"localhost"`

	Debug  bool     `env:"DEBUG"  default:"false"`
	APIKey string   `secret:"API_KEY"`
	Tags   []string `env:"TAGS"   default:"web,api"`
    
    DefaultMem  resource.Quantity `env:"DEFAULT_MEM" default:"1Gi"`
	DefaultDisk resource.Quantity `env:"DEFAULT_DISK" default:"10G"` 
    
    IPAddress net.IP        `env:"IP_ADDRESS"`
	EmailAddr mail.Address  `env:"EMAIL_ADDR"`

    LogLevel      slog.Level   `env:"LOG_LEVEL"`

    Database struct {
        Host string `env:"DB_HOST" default:"localhost"`
        Port int    `env:"DB_PORT" default:"5432"`
    }

    DatabaseURL url.URL       `env:"DB_URL"`
}

func main() {
	cfg, err := gonfig.Load(Config{})

	fmt.Println(gonfig.PrettyString(cfg)) // secrets masked
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

## API

```go
func Load[T any](cfg *T) *T
func LoadWithDotenv[T any](cfg *T, paths ...string) *T
func PrettyString(v any) string
```

## License

This project is licensed under the MIT License.