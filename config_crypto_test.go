package getconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"
)

// Test RSA private key parsing from PEM format
func TestRSAPrivateKey(t *testing.T) {
	// Generate a test RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA private key: %v", err)
	}

	// Convert to PEM format (PKCS#1)
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	tests := []struct {
		name      string
		configPtr interface{}
		envVar    string
		pemData   string
		wantErr   bool
	}{
		{
			name: "RSA private key pointer",
			configPtr: &struct {
				Key *rsa.PrivateKey `env:"RSA_KEY"`
			}{},
			envVar:  "RSA_KEY",
			pemData: string(privateKeyPEM),
			wantErr: false,
		},
		{
			name: "RSA private key value",
			configPtr: &struct {
				Key rsa.PrivateKey `env:"RSA_KEY_VALUE"`
			}{},
			envVar:  "RSA_KEY_VALUE",
			pemData: string(privateKeyPEM),
			wantErr: false,
		},
		{
			name: "Invalid PEM format",
			configPtr: &struct {
				Key *rsa.PrivateKey `env:"RSA_KEY_INVALID"`
			}{},
			envVar:  "RSA_KEY_INVALID",
			pemData: "invalid pem data",
			wantErr: true,
		},
		{
			name: "Empty PEM",
			configPtr: &struct {
				Key *rsa.PrivateKey `env:"RSA_KEY_EMPTY"`
			}{},
			envVar:  "RSA_KEY_EMPTY",
			pemData: "-----BEGIN RSA PRIVATE KEY-----\n-----END RSA PRIVATE KEY-----",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv(tt.envVar, tt.pemData)
			defer os.Unsetenv(tt.envVar)

			// Load into provided config pointer, ignore returned value
			_, err := Load(tt.configPtr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If no error expected, verify the key was loaded correctly
			if !tt.wantErr {
				switch cfg := tt.configPtr.(type) {
				case *struct{ Key *rsa.PrivateKey }:
					if cfg.Key == nil {
						t.Error("Expected RSA private key to be loaded, got nil")
					} else if cfg.Key.N.Cmp(privateKey.N) != 0 {
						t.Error("Loaded RSA key modulus doesn't match original")
					}
				case *struct{ Key rsa.PrivateKey }:
					if cfg.Key.N.Cmp(privateKey.N) != 0 {
						t.Error("Loaded RSA key modulus doesn't match original")
					}
				}
			}
		})
	}
}

// Test RSA private key with PKCS#8 format
func TestRSAPrivateKeyPKCS8(t *testing.T) {
	// Generate a test RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA private key: %v", err)
	}

	// Convert to PKCS#8 PEM format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal PKCS#8 private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	config := &struct {
		Key *rsa.PrivateKey `env:"RSA_PKCS8_KEY"`
	}{}

	// Set environment variable
	os.Setenv("RSA_PKCS8_KEY", string(privateKeyPEM))
	defer os.Unsetenv("RSA_PKCS8_KEY")

	// Load into provided config pointer and capture error
	// Load into provided config pointer and capture error
	_, err = Load(config)
	if err != nil {
		t.Errorf("Load() error = %v, expected no error", err)
		return
	}

	if config.Key == nil {
		t.Error("Expected RSA private key to be loaded, got nil")
	} else if config.Key.N.Cmp(privateKey.N) != 0 {
		t.Error("Loaded RSA key modulus doesn't match original")
	}
}

// Test ECDSA private key parsing from PEM format
func TestECDSAPrivateKey(t *testing.T) {
	// Generate a test ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA private key: %v", err)
	}

	// Convert to PEM format
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	tests := []struct {
		name      string
		configPtr interface{}
		envVar    string
		pemData   string
		wantErr   bool
	}{
		{
			name: "ECDSA private key pointer",
			configPtr: &struct {
				Key *ecdsa.PrivateKey `env:"ECDSA_KEY"`
			}{},
			envVar:  "ECDSA_KEY",
			pemData: string(privateKeyPEM),
			wantErr: false,
		},
		{
			name: "ECDSA private key value",
			configPtr: &struct {
				Key ecdsa.PrivateKey `env:"ECDSA_KEY_VALUE"`
			}{},
			envVar:  "ECDSA_KEY_VALUE",
			pemData: string(privateKeyPEM),
			wantErr: false,
		},
		{
			name: "Invalid PEM format",
			configPtr: &struct {
				Key *ecdsa.PrivateKey `env:"ECDSA_KEY_INVALID"`
			}{},
			envVar:  "ECDSA_KEY_INVALID",
			pemData: "invalid pem data",
			wantErr: true,
		},
		{
			name: "Empty PEM",
			configPtr: &struct {
				Key *ecdsa.PrivateKey `env:"ECDSA_KEY_EMPTY"`
			}{},
			envVar:  "ECDSA_KEY_EMPTY",
			pemData: "-----BEGIN EC PRIVATE KEY-----\n-----END EC PRIVATE KEY-----",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv(tt.envVar, tt.pemData)
			defer os.Unsetenv(tt.envVar)

			_, err := Load(tt.configPtr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If no error expected, verify the key was loaded correctly
			if !tt.wantErr {
				switch cfg := tt.configPtr.(type) {
				case *struct{ Key *ecdsa.PrivateKey }:
					if cfg.Key == nil {
						t.Error("Expected ECDSA private key to be loaded, got nil")
					} else if cfg.Key.X.Cmp(privateKey.X) != 0 || cfg.Key.Y.Cmp(privateKey.Y) != 0 {
						t.Error("Loaded ECDSA key doesn't match original")
					}
				case *struct{ Key ecdsa.PrivateKey }:
					if cfg.Key.X.Cmp(privateKey.X) != 0 || cfg.Key.Y.Cmp(privateKey.Y) != 0 {
						t.Error("Loaded ECDSA key doesn't match original")
					}
				}
			}
		})
	}
}

// Test ECDSA private key with PKCS#8 format
func TestECDSAPrivateKeyPKCS8(t *testing.T) {
	// Generate a test ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA private key: %v", err)
	}

	// Convert to PKCS#8 PEM format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal PKCS#8 private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	config := &struct {
		Key *ecdsa.PrivateKey `env:"ECDSA_PKCS8_KEY"`
	}{}

	// Set environment variable
	os.Setenv("ECDSA_PKCS8_KEY", string(privateKeyPEM))
	defer os.Unsetenv("ECDSA_PKCS8_KEY")

	_, err = Load(config)
	if err != nil {
		t.Errorf("Load() error = %v, expected no error", err)
		return
	}

	if config.Key == nil {
		t.Error("Expected ECDSA private key to be loaded, got nil")
	} else if config.Key.X.Cmp(privateKey.X) != 0 || config.Key.Y.Cmp(privateKey.Y) != 0 {
		t.Error("Loaded ECDSA key doesn't match original")
	}
}

// Test mixed key types (RSA and ECDSA in same config)
func TestMixedPrivateKeys(t *testing.T) {
	// Generate test keys
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA private key: %v", err)
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA private key: %v", err)
	}

	// Convert to PEM formats
	rsaBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
	rsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: rsaBytes,
	})

	ecdsaBytes, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA private key: %v", err)
	}
	ecdsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: ecdsaBytes,
	})

	config := &struct {
		RSAKey   *rsa.PrivateKey   `secret:"JWT_RSA_KEY"`
		ECDSAKey *ecdsa.PrivateKey `secret:"JWT_ECDSA_KEY"`
		AppName  string            `env:"APP_NAME" default:"test-app"`
	}{}

	// Set environment variables
	os.Setenv("JWT_RSA_KEY", string(rsaPEM))
	os.Setenv("JWT_ECDSA_KEY", string(ecdsaPEM))
	defer func() {
		os.Unsetenv("JWT_RSA_KEY")
		os.Unsetenv("JWT_ECDSA_KEY")
	}()

	_, err = Load(config)
	if err != nil {
		t.Errorf("Load() error = %v, expected no error", err)
		return
	}

	if config.RSAKey == nil {
		t.Error("Expected RSA private key to be loaded, got nil")
	} else if config.RSAKey.N.Cmp(rsaKey.N) != 0 {
		t.Error("Loaded RSA key modulus doesn't match original")
	}

	if config.ECDSAKey == nil {
		t.Error("Expected ECDSA private key to be loaded, got nil")
	} else if config.ECDSAKey.X.Cmp(ecdsaKey.X) != 0 || config.ECDSAKey.Y.Cmp(ecdsaKey.Y) != 0 {
		t.Error("Loaded ECDSA key doesn't match original")
	}

	if config.AppName != "test-app" {
		t.Errorf("Expected AppName to be 'test-app', got %s", config.AppName)
	}
}

// Test PrettyString masking for private keys
func TestPrettyStringPrivateKeys(t *testing.T) {
	// Generate test keys
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA private key: %v", err)
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA private key: %v", err)
	}

	// Convert to PEM formats
	rsaBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
	rsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: rsaBytes,
	})

	ecdsaBytes, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA private key: %v", err)
	}
	ecdsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: ecdsaBytes,
	})

	config := &struct {
		RSAKey   *rsa.PrivateKey   `secret:"JWT_RSA_KEY"`
		ECDSAKey *ecdsa.PrivateKey `secret:"JWT_ECDSA_KEY"`
		AppName  string            `env:"APP_NAME" default:"test-app"`
	}{}

	// Set environment variables and load
	os.Setenv("JWT_RSA_KEY", string(rsaPEM))
	os.Setenv("JWT_ECDSA_KEY", string(ecdsaPEM))
	defer func() {
		os.Unsetenv("JWT_RSA_KEY")
		os.Unsetenv("JWT_ECDSA_KEY")
	}()

	_, err = Load(config)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test PrettyString output
	prettyStr := PrettyString(config)

	// Verify that private keys are masked as "***"
	if !containsString(prettyStr, `"JWT_RSA_KEY": "***"`) {
		t.Errorf("Expected RSA private key to be masked as ***, got: %s", prettyStr)
	}

	if !containsString(prettyStr, `"JWT_ECDSA_KEY": "***"`) {
		t.Errorf("Expected ECDSA private key to be masked as ***, got: %s", prettyStr)
	}

	if !containsString(prettyStr, `"APP_NAME": "test-app"`) {
		t.Errorf("Expected APP_NAME to be visible, got: %s", prettyStr)
	}
}

// Test error cases for wrong key types in PKCS#8
func TestPKCS8WrongKeyType(t *testing.T) {
	// Generate an ECDSA key but try to parse it as RSA
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA private key: %v", err)
	}

	// Convert to PKCS#8 PEM format
	ecdsaBytes, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
	if err != nil {
		t.Fatalf("Failed to marshal PKCS#8 private key: %v", err)
	}
	ecdsaPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: ecdsaBytes,
	})

	// Try to load as RSA key
	config := &struct {
		Key *rsa.PrivateKey `env:"WRONG_KEY_TYPE"`
	}{}

	os.Setenv("WRONG_KEY_TYPE", string(ecdsaPEM))
	defer os.Unsetenv("WRONG_KEY_TYPE")

	_, err = Load(config)
	if err == nil {
		t.Error("Expected error when loading ECDSA key as RSA, got nil")
	} else if !containsString(err.Error(), "PKCS#8 key is not an RSA private key") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
