// example.go
//
// A single, fully-featured configuration struct exercising every capability
// of the gonfig library. Copy this file into your project, run
//
//     go run example.go
//
// and play with environment variables or a “.env” file to see how the
// values change. Secrets are always masked in PrettyString output.

package main

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/mail"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/veiloq/gonfig"
	"k8s.io/apimachinery/pkg/api/resource"
)

/*
 * Custom type example (implements encoding.TextUnmarshaler)
 */
type Color string

func (c *Color) UnmarshalText(text []byte) error {
	*c = Color(text)
	return nil
}

/*
 * Nested struct example
 */
type Credentials struct {
	Username string `env:"USERNAME"        default:"guest"`
	Password string `secret:"PASSWORD"` // masked in logs
}

/*
 * One gigantic struct covering every supported field type
 */
type AppConfig struct {
	// Basic types
	AppName string  `env:"APP_NAME"  default:"gonfig-app"`
	Port    int     `env:"PORT"      default:"8080"`
	Debug   bool    `env:"DEBUG"     default:"false"`
	Pi      float64 `env:"PI"        default:"3.141592"`

	// CSV slices
	Tags   []string  `env:"TAGS"   default:"api,web,service"`
	Ports  []int     `env:"PORTS"  default:"8080,8081"`
	Ratios []float64 `env:"RATIOS" default:"1.0,2.5,3.14"`
	Flags  []bool    `env:"FLAGS"  default:"true,false,true"`

	// Time-related
	Timeout   time.Duration `env:"TIMEOUT"     default:"30s"`
	StartTime time.Time     `env:"START_TIME"  default:"2025-01-01T15:04:05Z"`

	// Logging level
	LogLevel slog.Level `env:"LOG_LEVEL" default:"info"`

	// Extended numeric
	BigCount *big.Int        `env:"BIG_COUNT" default:"12345678901234567890"`
	Price    decimal.Decimal `env:"PRICE"    default:"19.99"`

	// Networking and URLs
	DatabaseURL url.URL `env:"DATABASE_URL" default:"postgres://user:pass@localhost:5432/db"`
	ListenIP    net.IP  `env:"LISTEN_IP"    default:"127.0.0.1"`

	// E-mail, UUID, K8s resource quantity
	AdminEmail mail.Address      `env:"ADMIN_EMAIL" default:"Admin <admin@example.com>"`
	InstanceID uuid.UUID         `env:"INSTANCE_ID" default:"00000000-0000-0000-0000-000000000000"`
	Memory     resource.Quantity `env:"MEMORY"      default:"512Mi"`

	// Cryptographic keys (PEM)
	RSAKey *rsa.PrivateKey   `secret:"RSA_PRIVATE_KEY"`
	ECKey  *ecdsa.PrivateKey `secret:"EC_PRIVATE_KEY"`

	// Secrets and secret slices
	APIKey  string   `secret:"API_KEY"`
	APIKeys []string `secret:"API_KEYS" default:"secret1,secret2"`

	// Custom color
	Theme Color `env:"THEME" default:"light"`

	// Nested struct
	Creds Credentials
}

func main() {
	cfg := &AppConfig{}

	// Load from env, optional .env file, and defaults (in that order)
	gonfig.LoadWithDotenv(cfg)

	fmt.Println("Current configuration (secrets masked):")
	fmt.Println(gonfig.PrettyString(cfg))

	fmt.Printf("▶ Starting %s on %s:%d (debug=%v)…\n",
		cfg.AppName, cfg.ListenIP, cfg.Port, cfg.Debug)
}
