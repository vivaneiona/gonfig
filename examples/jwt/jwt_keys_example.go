package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	getconfig "github.com/vivaneiona/gonfig"
)

// JWTConfig represents a configuration for JWT signing
type JWTConfig struct {
	AppName       string            `env:"APP_NAME" default:"jwt-service"`
	Port          int               `env:"PORT" default:"8080"`
	RSASigningKey *rsa.PrivateKey   `secret:"JWT_RSA_PRIVATE_KEY"`
	ECSigningKey  *ecdsa.PrivateKey `secret:"JWT_EC_PRIVATE_KEY"`
}

func main() {
	// Generate fresh RSA private key for demo
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Generate fresh ECDSA private key for demo
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	// Convert RSA key to PEM format
	rsaBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
	rsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: rsaBytes,
	})

	// Convert ECDSA key to PEM format
	ecdsaBytes, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		log.Fatalf("Failed to marshal ECDSA key: %v", err)
	}
	ecdsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: ecdsaBytes,
	})

	// Set environment variables for demo
	os.Setenv("JWT_RSA_PRIVATE_KEY", string(rsaPEM))
	os.Setenv("JWT_EC_PRIVATE_KEY", string(ecdsaPEM))

	// Load configuration
	// Initialize config struct and load into it
	config := &JWTConfig{}
	cfg, err := getconfig.Load(config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Verify keys were loaded
	fmt.Printf("App: %s\n", cfg.AppName)
	fmt.Printf("Port: %d\n", cfg.Port)

	if cfg.RSASigningKey != nil {
		fmt.Printf("RSA Key loaded: %d bits\n", cfg.RSASigningKey.N.BitLen())
	} else {
		fmt.Println("RSA Key: not loaded")
	}

	if cfg.ECSigningKey != nil {
		fmt.Printf("ECDSA Key loaded: %s curve\n", cfg.ECSigningKey.Curve.Params().Name)
	} else {
		fmt.Println("ECDSA Key: not loaded")
	}

	// Show safe configuration output (keys are masked)
	fmt.Println("\nSafe config output:")
	fmt.Println(getconfig.PrettyString(cfg))
}
