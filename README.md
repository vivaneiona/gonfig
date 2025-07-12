# ⚙︎ vivaneiona/gonfig

Go configuration library that loads configuration from environment variables with support for defaults, secret masking, and `.env` files.

## Features

- **Type-safe configuration loading** using Go generics
- **Multiple data types** supported: `string`, `bool`, `int`, `float64`, and slices thereof
- **CSV slice support** for comma-separated values in environment variables
- **Default values** via struct tags
- **Secret masking** for sensitive configuration values (including slices)
- **`.env` file support** with automatic loading
- **Pretty printing** with JSON output and masked secrets
- **Zero external dependencies** (except `godotenv` for `.env` support)

## Installation

```bash
go get github.com/vivaneiona/gonfig
```

## Quick Start

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/vivaneiona/gonfig"
)

type Config struct {
    Port     int      `env:"PORT" default:"8080"`
    Host     string   `env:"HOST" default:"localhost"`
    Debug    bool     `env:"DEBUG" default:"false"`
    APIKey   string   `secret:"API_KEY" default:""`
    Database string   `env:"DATABASE_URL" default:"sqlite://app.db"`
    Tags     []string `env:"TAGS" default:"web,api,service"`
    Ports    []int    `env:"PORTS" default:"8080,8081,8082"`
}

func main() {
    cfg := &Config{}
    config := getconfig.Load(cfg)
    
    fmt.Printf("Server starting on %s:%d\n", config.Host, config.Port)
    fmt.Printf("Debug mode: %v\n", config.Debug)
    
    // Pretty print configuration (secrets will be masked)
    fmt.Println("Configuration:")
    fmt.Println(getconfig.PrettyString(cfg))
}
```

## Struct Tags

The library uses struct tags to control configuration loading:

### `env` tag
Maps struct fields to environment variables:
```go
type Config struct {
    Port int `env:"PORT"`
}
```

### `secret` tag  
Maps struct fields to environment variables but masks them in output:
```go
type Config struct {
    APIKey string `secret:"API_KEY"`
}
```

### `default` tag
Provides fallback values when environment variables are not set:
```go
type Config struct {
    Port int `env:"PORT" default:"8080"`
}
```

## Supported Types

The library supports the following Go types:

### Basic Types
- `string`
- `bool` 
- `int`, `int8`, `int16`, `int32`, `int64`
- `float32`, `float64`
- `[]string`, `[]bool`, `[]int` (and other int types), `[]float32`, `[]float64` - for CSV values

### Extended Types
- `time.Duration` - parses duration strings like "5s", "1m30s", "2h45m"
- `time.Time` - parses RFC3339 format or Unix seconds
- `slog.Level` - parses "debug", "info", "warn", "error" or integer levels
- `big.Int` - parses base-10 integer strings for arbitrary precision
- `decimal.Decimal` - parses exact decimal arithmetic strings (github.com/shopspring/decimal)
- `url.URL` - parses URLs with automatic password masking in logs
  - Supports both TCP: `postgres://user:pass@host:port/db` 
  - And Unix sockets: `postgresql://user:pass@/db?host=/socket/path`
- `net.IP` - parses IPv4 and IPv6 addresses
- `mail.Address` - parses email addresses with optional display names
- `uuid.UUID` - parses UUID strings (github.com/google/uuid)
- `resource.Quantity` - parses Kubernetes resource units like "250m", "1.5Gi" (k8s.io/apimachinery)

### Cryptographic Types
- `rsa.PrivateKey` - parses RSA private keys from PEM format (PKCS#1 or PKCS#8)
- `ecdsa.PrivateKey` - parses ECDSA private keys from PEM format (EC or PKCS#8)

### Custom Types
- Any type implementing `encoding.TextUnmarshaler`

### Nested Structs
- Nested structs (value or pointer) with recursive processing

### CSV Slice Support

Slice fields automatically parse comma-separated values from environment variables:

```go
type Config struct {
    Tags     []string  `env:"TAGS" default:"web,api,service"`
    Ports    []int     `env:"PORTS" default:"8080,8081,8082"`
    Ratios   []float64 `env:"RATIOS" default:"1.0,2.5,3.14"`
    Features []bool    `env:"FEATURES" default:"true,false,true"`
}
```

Environment variables should contain comma-separated values:
```bash
export TAGS="production,backend,database"
export PORTS="3000,4000,5000"
export RATIOS="0.5,1.5,2.0"
export FEATURES="true,true,false"
```

## API Reference

### `Load[T any](config T) T`

Loads configuration from environment variables into the provided struct pointer.

**Parameters:**
- `config`: Pointer to a struct with tagged fields

**Returns:**
- The same struct with fields populated from environment variables

**Example:**
```go
cfg := &Config{}
loaded := getconfig.Load(cfg)
```

### `LoadWithDotenv[T any](config T, dotenvPath ...string) T`

Loads configuration from environment variables with optional `.env` file support.

**Parameters:**
- `config`: Pointer to a struct with tagged fields  
- `dotenvPath`: Optional path to `.env` file (defaults to `.env`)

**Returns:**
- The same struct with fields populated from environment variables and `.env` file

**Example:**
```go
cfg := &Config{}
loaded := getconfig.LoadWithDotenv(cfg, "config/.env")
```

### `PrettyString(c any) string`

Returns a JSON-formatted string representation of the configuration with secrets masked.

**Parameters:**
- `c`: Pointer to a configuration struct

**Returns:**
- JSON string with configuration values (secrets masked)

**Example:**
```go
cfg := &Config{APIKey: "secret123"}
fmt.Println(getconfig.PrettyString(cfg))
// Output: {"API_KEY": "sec****", ...}
```

## Secret Masking

Fields tagged with `secret` are automatically masked in the output of `PrettyString()`. The masking shows the first 3 characters followed by asterisks. This also works for slices of strings:

**String secrets:**
- `""` → `""`
- `"a"` → `"*"`  
- `"ab"` → `"**"`
- `"abc"` → `"***"`
- `"abcd"` → `"abc*"`
- `"secret123"` → `"sec*****"`

**Slice secrets:**
```go
type Config struct {
    APIKeys []string `secret:"API_KEYS"`
}
// If API_KEYS="secret1,secret2,secret3"
// PrettyString output: {"API_KEYS": ["sec****", "sec****", "sec****"]}
```

## Environment Variable Loading

The library follows this precedence order:

1. **Environment variables** (highest priority)
2. **`.env` file values** (when using `LoadWithDotenv`)
3. **Default tag values** (lowest priority)

## Examples

### Basic Configuration

```go
type AppConfig struct {
    Port    int    `env:"PORT" default:"3000"`
    Host    string `env:"HOST" default:"0.0.0.0"`
    Debug   bool   `env:"DEBUG" default:"false"`
}

cfg := &AppConfig{}
config := getconfig.Load(cfg)
```

### With CSV Slices

```go
type ServerConfig struct {
    Host        string    `env:"HOST" default:"localhost"`
    Ports       []int     `env:"PORTS" default:"8080,8081,8082"`
    AllowedIPs  []string  `env:"ALLOWED_IPS" default:"127.0.0.1,::1"`
    Features    []bool    `env:"FEATURES" default:"true,false,true"`
    Ratios      []float64 `env:"RATIOS" default:"1.0,2.5,3.14"`
    SecretKeys  []string  `secret:"SECRET_KEYS"`
}

cfg := &ServerConfig{}
config := getconfig.Load(cfg)

// Access slice values
fmt.Printf("Listening on ports: %v\n", config.Ports)
fmt.Printf("Allowed IPs: %v\n", config.AllowedIPs)

// Safe to log - secret slices will be masked
fmt.Println(getconfig.PrettyString(cfg))
```

Set environment variables:
```bash
export PORTS="3000,4000,5000"
export ALLOWED_IPS="192.168.1.1,10.0.0.1"
export FEATURES="true,true,false"
export SECRET_KEYS="key1,key2,key3"
```

### With Secrets

```go
type DatabaseConfig struct {
    Host     string `env:"DB_HOST" default:"localhost"`
    Port     int    `env:"DB_PORT" default:"5432"`
    Username string `env:"DB_USER" default:"postgres"`
    Password string `secret:"DB_PASSWORD"`
    SSLMode  string `env:"DB_SSLMODE" default:"disable"`
}

cfg := &DatabaseConfig{}
config := getconfig.LoadWithDotenv(cfg)

// Safe to log - password will be masked
fmt.Println(getconfig.PrettyString(cfg))
```

### Using .env Files

Create a `.env` file:
```env
PORT=8080
DEBUG=true
API_KEY=your-secret-key
DATABASE_URL=postgres://user:pass@localhost/db
SOCKET_DB_URL=postgresql://user:pass@/db?host=/var/run/postgresql
```

Load configuration:
```go
type Config struct {
    Port        int    `env:"PORT" default:"3000"`
    Debug       bool   `env:"DEBUG" default:"false"`
    APIKey      string `secret:"API_KEY"`
    DatabaseURL string `env:"DATABASE_URL"`
    SocketDBURL string `env:"SOCKET_DB_URL"` // PostgreSQL Unix socket
}

cfg := &Config{}
config := getconfig.LoadWithDotenv(cfg)
```

### JWT Configuration with Cryptographic Keys

```go
import (
    "crypto/ecdsa"
    "crypto/rsa"
    "github.com/vivaneiona/gonfig"
)

type JWTConfig struct {
    AppName        string             `env:"APP_NAME" default:"jwt-service"`
    Port          int                `env:"PORT" default:"8080"`
    RSASigningKey *rsa.PrivateKey    `secret:"JWT_RSA_PRIVATE_KEY"`
    ECSigningKey  *ecdsa.PrivateKey  `secret:"JWT_EC_PRIVATE_KEY"`
    TokenIssuer   string             `env:"JWT_ISSUER" default:"my-service"`
}

cfg := &JWTConfig{}
config := getconfig.Load(cfg)

// Private keys are automatically parsed from PEM format
// Both PKCS#1 ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE KEY") formats supported
// Keys are masked in PrettyString output for security
```

Environment variables should contain PEM-encoded private keys:
```bash
export JWT_RSA_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"

export JWT_EC_PRIVATE_KEY="-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIG2c3eLL...
-----END EC PRIVATE KEY-----"
```

## Error Handling

The library is designed to be forgiving:

- Missing environment variables fall back to default values
- Invalid type conversions are ignored (field keeps zero value)
- Missing `.env` files are silently ignored
- Unsupported field types cause a panic (fail-fast for development)

## License

This project is licensed under the MIT License.