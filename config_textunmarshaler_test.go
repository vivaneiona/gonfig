package getconfig

import (
	"net"
	"net/mail"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// textUnmarshalerTestConfig demonstrates types that implement encoding.TextUnmarshaler
type textUnmarshalerTestConfig struct {
	// Standard library types that implement TextUnmarshaler
	IPAddress net.IP        `env:"IP_ADDRESS"`
	EmailAddr mail.Address  `env:"EMAIL_ADDR"`
	UUID      uuid.UUID     `env:"UUID"`
	Duration  time.Duration `env:"DURATION"` // Also implements TextUnmarshaler
	URL       url.URL       `env:"URL"`      // Also implements TextUnmarshaler

	// Pointer versions
	IPAddressPtr *net.IP       `env:"IP_ADDRESS_PTR"`
	EmailAddrPtr *mail.Address `env:"EMAIL_ADDR_PTR"`
	UUIDPtr      *uuid.UUID    `env:"UUID_PTR"`

	// Lists
	IPList    []net.IP       `env:"IP_LIST"`
	UUIDList  []uuid.UUID    `env:"UUID_LIST"`
	EmailList []mail.Address `env:"EMAIL_LIST"`
}

func TestTextUnmarshalerIP(t *testing.T) {
	// Test net.IP parsing via TextUnmarshaler
	os.Setenv("IP_ADDRESS", "192.168.1.1")
	defer os.Unsetenv("IP_ADDRESS")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := net.ParseIP("192.168.1.1")
	if !cfg.IPAddress.Equal(expected) {
		t.Errorf("IPAddress = %v; want %v", cfg.IPAddress, expected)
	}
}

func TestTextUnmarshalerIPv6(t *testing.T) {
	// Test IPv6 parsing
	os.Setenv("IP_ADDRESS", "2001:db8::1")
	defer os.Unsetenv("IP_ADDRESS")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := net.ParseIP("2001:db8::1")
	if !cfg.IPAddress.Equal(expected) {
		t.Errorf("IPAddress = %v; want %v", cfg.IPAddress, expected)
	}
}

func TestTextUnmarshalerEmail(t *testing.T) {
	// Test mail.Address parsing
	os.Setenv("EMAIL_ADDR", "test@example.com")
	defer os.Unsetenv("EMAIL_ADDR")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.EmailAddr.Address != "test@example.com" {
		t.Errorf("EmailAddr.Address = %v; want %v", cfg.EmailAddr.Address, "test@example.com")
	}
}

func TestTextUnmarshalerEmailWithName(t *testing.T) {
	// Test mail.Address with name
	os.Setenv("EMAIL_ADDR", "Test User <test@example.com>")
	defer os.Unsetenv("EMAIL_ADDR")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.EmailAddr.Name != "Test User" {
		t.Errorf("EmailAddr.Name = %v; want %v", cfg.EmailAddr.Name, "Test User")
	}
	if cfg.EmailAddr.Address != "test@example.com" {
		t.Errorf("EmailAddr.Address = %v; want %v", cfg.EmailAddr.Address, "test@example.com")
	}
}

func TestTextUnmarshalerUUID(t *testing.T) {
	// Test UUID parsing
	uuidStr := "123e4567-e89b-12d3-a456-426614174000"
	os.Setenv("UUID", uuidStr)
	defer os.Unsetenv("UUID")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Fatalf("Failed to parse expected UUID: %v", err)
	}

	if cfg.UUID != expected {
		t.Errorf("UUID = %v; want %v", cfg.UUID, expected)
	}
}

func TestTextUnmarshalerDuration(t *testing.T) {
	// Test time.Duration parsing (should use explicit parser, not TextUnmarshaler)
	os.Setenv("DURATION", "5m30s")
	defer os.Unsetenv("DURATION")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := 5*time.Minute + 30*time.Second
	if cfg.Duration != expected {
		t.Errorf("Duration = %v; want %v", cfg.Duration, expected)
	}
}

func TestTextUnmarshalerURL(t *testing.T) {
	// Test url.URL parsing (should use explicit parser, not TextUnmarshaler)
	urlStr := "https://example.com/path?query=value"
	os.Setenv("URL", urlStr)
	defer os.Unsetenv("URL")

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, err := url.Parse(urlStr)
	if err != nil {
		t.Fatalf("Failed to parse expected URL: %v", err)
	}

	if cfg.URL.String() != expected.String() {
		t.Errorf("URL = %v; want %v", cfg.URL, *expected)
	}
}

func TestTextUnmarshalerPointers(t *testing.T) {
	// Test pointer types
	os.Setenv("IP_ADDRESS_PTR", "10.0.0.1")
	os.Setenv("EMAIL_ADDR_PTR", "ptr@example.com")
	os.Setenv("UUID_PTR", "550e8400-e29b-41d4-a716-446655440000")
	defer func() {
		os.Unsetenv("IP_ADDRESS_PTR")
		os.Unsetenv("EMAIL_ADDR_PTR")
		os.Unsetenv("UUID_PTR")
	}()

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check IP pointer
	if cfg.IPAddressPtr == nil {
		t.Fatal("IPAddressPtr should not be nil")
	}
	expectedIP := net.ParseIP("10.0.0.1")
	if !cfg.IPAddressPtr.Equal(expectedIP) {
		t.Errorf("IPAddressPtr = %v; want %v", *cfg.IPAddressPtr, expectedIP)
	}

	// Check email pointer
	if cfg.EmailAddrPtr == nil {
		t.Fatal("EmailAddrPtr should not be nil")
	}
	if cfg.EmailAddrPtr.Address != "ptr@example.com" {
		t.Errorf("EmailAddrPtr.Address = %v; want %v", cfg.EmailAddrPtr.Address, "ptr@example.com")
	}

	// Check UUID pointer
	if cfg.UUIDPtr == nil {
		t.Fatal("UUIDPtr should not be nil")
	}
	expectedUUID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	if *cfg.UUIDPtr != expectedUUID {
		t.Errorf("UUIDPtr = %v; want %v", *cfg.UUIDPtr, expectedUUID)
	}
}

func TestTextUnmarshalerLists(t *testing.T) {
	// Test lists of TextUnmarshaler types
	os.Setenv("IP_LIST", "192.168.1.1,10.0.0.1,::1")
	os.Setenv("UUID_LIST", "123e4567-e89b-12d3-a456-426614174000,550e8400-e29b-41d4-a716-446655440000")
	os.Setenv("EMAIL_LIST", "user1@example.com,User Two <user2@example.com>")
	defer func() {
		os.Unsetenv("IP_LIST")
		os.Unsetenv("UUID_LIST")
		os.Unsetenv("EMAIL_LIST")
	}()

	cfg, err := Load(textUnmarshalerTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check IP list
	expectedIPs := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("10.0.0.1"),
		net.ParseIP("::1"),
	}
	if len(cfg.IPList) != len(expectedIPs) {
		t.Fatalf("IPList length = %d; want %d", len(cfg.IPList), len(expectedIPs))
	}
	for i, ip := range cfg.IPList {
		if !ip.Equal(expectedIPs[i]) {
			t.Errorf("IPList[%d] = %v; want %v", i, ip, expectedIPs[i])
		}
	}

	// Check UUID list
	expectedUUIDs := []uuid.UUID{
		uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
	}
	if len(cfg.UUIDList) != len(expectedUUIDs) {
		t.Fatalf("UUIDList length = %d; want %d", len(cfg.UUIDList), len(expectedUUIDs))
	}
	for i, u := range cfg.UUIDList {
		if u != expectedUUIDs[i] {
			t.Errorf("UUIDList[%d] = %v; want %v", i, u, expectedUUIDs[i])
		}
	}

	// Check email list
	if len(cfg.EmailList) != 2 {
		t.Fatalf("EmailList length = %d; want 2", len(cfg.EmailList))
	}
	if cfg.EmailList[0].Address != "user1@example.com" {
		t.Errorf("EmailList[0].Address = %v; want %v", cfg.EmailList[0].Address, "user1@example.com")
	}
	if cfg.EmailList[1].Name != "User Two" || cfg.EmailList[1].Address != "user2@example.com" {
		t.Errorf("EmailList[1] = %v; want Name='User Two', Address='user2@example.com'", cfg.EmailList[1])
	}
}

func TestTextUnmarshalerInvalidIP(t *testing.T) {
	// Test invalid IP address
	os.Setenv("IP_ADDRESS", "invalid-ip")
	defer os.Unsetenv("IP_ADDRESS")

	_, err := Load(textUnmarshalerTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid IP address")
	}
}

func TestTextUnmarshalerInvalidUUID(t *testing.T) {
	// Test invalid UUID
	os.Setenv("UUID", "invalid-uuid")
	defer os.Unsetenv("UUID")

	_, err := Load(textUnmarshalerTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid UUID")
	}
}

func TestTextUnmarshalerInvalidEmail(t *testing.T) {
	// Test invalid email address
	os.Setenv("EMAIL_ADDR", "invalid-email")
	defer os.Unsetenv("EMAIL_ADDR")

	_, err := Load(textUnmarshalerTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid email address")
	}
}
