# ⚙︎ vivaneiona/gonfig

[![Go Reference](https://pkg.go.dev/badge/github.com/vivaneiona/gonfig.svg)](https://pkg.go.dev/github.com/vivaneiona/gonfig)

*A small, type-safe configuration loader for Go.*

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

## API

```go
func Load[T any](cfg *T) *T
func LoadWithDotenv[T any](cfg *T, paths ...string) *T
func PrettyString(v any) string
```

## License

This project is licensed under the MIT License.