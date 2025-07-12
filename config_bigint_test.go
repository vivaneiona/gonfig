package gonfig

import (
	"math/big"
	"os"
	"testing"
)

// bigIntTestConfig is used to test big.Int parsing
type bigIntTestConfig struct {
	BigInt        big.Int    `env:"BIG_INT"`
	BigIntPtr     *big.Int   `env:"BIG_INT_PTR"`
	BigIntList    []big.Int  `env:"BIG_INT_LIST"`
	BigIntPtrList []*big.Int `env:"BIG_INT_PTR_LIST"`
	SmallInt      big.Int    `env:"SMALL_INT" default:"42"`
	LargeInt      big.Int    `env:"LARGE_INT" default:"123456789012345678901234567890"`
}

func TestBigIntValue(t *testing.T) {
	// Test big.Int value parsing
	os.Setenv("BIG_INT", "123456789012345678901234567890")
	defer os.Unsetenv("BIG_INT")

	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := new(big.Int)
	expected.SetString("123456789012345678901234567890", 10)

	if cfg.BigInt.Cmp(expected) != 0 {
		t.Errorf("BigInt = %v; want %v", &cfg.BigInt, expected)
	}
}

func TestBigIntPtr(t *testing.T) {
	// Test *big.Int parsing
	os.Setenv("BIG_INT_PTR", "987654321098765432109876543210")
	defer os.Unsetenv("BIG_INT_PTR")
	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.BigIntPtr == nil {
		t.Fatal("BigIntPtr should not be nil")
	}

	expected := new(big.Int)
	expected.SetString("987654321098765432109876543210", 10)

	if cfg.BigIntPtr.Cmp(expected) != 0 {
		t.Errorf("BigIntPtr = %v; want %v", cfg.BigIntPtr, expected)
	}
}

func TestBigIntNegative(t *testing.T) {
	// Test negative big.Int
	os.Setenv("BIG_INT", "-123456789012345678901234567890")
	defer os.Unsetenv("BIG_INT")
	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := new(big.Int)
	expected.SetString("-123456789012345678901234567890", 10)

	if cfg.BigInt.Cmp(expected) != 0 {
		t.Errorf("BigInt = %v; want %v", &cfg.BigInt, expected)
	}
}

func TestBigIntZero(t *testing.T) {
	// Test zero big.Int
	os.Setenv("BIG_INT", "0")
	defer os.Unsetenv("BIG_INT")
	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := big.NewInt(0)

	if cfg.BigInt.Cmp(expected) != 0 {
		t.Errorf("BigInt = %v; want %v", &cfg.BigInt, expected)
	}
}

func TestBigIntInvalid(t *testing.T) {
	// Test invalid big.Int
	os.Setenv("BIG_INT", "not-a-number")
	defer os.Unsetenv("BIG_INT")
	_, err := Load(bigIntTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid big.Int")
	}
}

func TestBigIntInvalidHex(t *testing.T) {
	// Test hex input (should fail since we only accept base-10)
	os.Setenv("BIG_INT", "0x123")
	defer os.Unsetenv("BIG_INT")
	_, err := Load(bigIntTestConfig{})
	if err == nil {
		t.Error("Load should have failed with hex big.Int (only base-10 allowed)")
	}
}

func TestBigIntList(t *testing.T) {
	// Test []big.Int parsing
	os.Setenv("BIG_INT_LIST", "123,456789012345678901234567890,-789")
	defer os.Unsetenv("BIG_INT_LIST")
	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"123", "456789012345678901234567890", "-789"}
	if len(cfg.BigIntList) != len(expectedValues) {
		t.Fatalf("BigIntList length = %d; want %d", len(cfg.BigIntList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		expected := new(big.Int)
		expected.SetString(expectedStr, 10)

		if cfg.BigIntList[i].Cmp(expected) != 0 {
			t.Errorf("BigIntList[%d] = %v; want %v", i, &cfg.BigIntList[i], expected)
		}
	}
}

func TestBigIntPtrList(t *testing.T) {
	// Test []*big.Int parsing
	os.Setenv("BIG_INT_PTR_LIST", "111,222333444555666777888999000")
	defer os.Unsetenv("BIG_INT_PTR_LIST")

	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"111", "222333444555666777888999000"}
	if len(cfg.BigIntPtrList) != len(expectedValues) {
		t.Fatalf("BigIntPtrList length = %d; want %d", len(cfg.BigIntPtrList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		if cfg.BigIntPtrList[i] == nil {
			t.Fatalf("BigIntPtrList[%d] should not be nil", i)
		}

		expected := new(big.Int)
		expected.SetString(expectedStr, 10)

		if cfg.BigIntPtrList[i].Cmp(expected) != 0 {
			t.Errorf("BigIntPtrList[%d] = %v; want %v", i, cfg.BigIntPtrList[i], expected)
		}
	}
}

func TestBigIntDefaults(t *testing.T) {
	// Test default values for big.Int
	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check small int default
	expected := big.NewInt(42)
	if cfg.SmallInt.Cmp(expected) != 0 {
		t.Errorf("SmallInt = %v; want %v", &cfg.SmallInt, expected)
	}

	// Check large int default
	expectedLarge := new(big.Int)
	expectedLarge.SetString("123456789012345678901234567890", 10)
	if cfg.LargeInt.Cmp(expectedLarge) != 0 {
		t.Errorf("LargeInt = %v; want %v", &cfg.LargeInt, expectedLarge)
	}
}

func TestBigIntRequired(t *testing.T) {
	// Test required big.Int field
	type bigIntRequiredConfig struct {
		BigInt big.Int `env:"REQUIRED_BIG_INT" required:"true"`
	}

	_, err := Load(bigIntRequiredConfig{})
	if err == nil {
		t.Error("Load should have failed with missing required big.Int")
	}
}

func TestBigIntEmptyList(t *testing.T) {
	// Test empty big.Int list
	os.Setenv("BIG_INT_LIST", "")
	defer os.Unsetenv("BIG_INT_LIST")

	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.BigIntList) != 0 {
		t.Errorf("BigIntList length = %d; want 0", len(cfg.BigIntList))
	}
}

func TestBigIntSpacesInList(t *testing.T) {
	// Test big.Int list with spaces
	os.Setenv("BIG_INT_LIST", " 123 , 456 , 789 ")
	defer os.Unsetenv("BIG_INT_LIST")

	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"123", "456", "789"}
	if len(cfg.BigIntList) != len(expectedValues) {
		t.Fatalf("BigIntList length = %d; want %d", len(cfg.BigIntList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		expected := new(big.Int)
		expected.SetString(expectedStr, 10)

		if cfg.BigIntList[i].Cmp(expected) != 0 {
			t.Errorf("BigIntList[%d] = %v; want %v", i, &cfg.BigIntList[i], expected)
		}
	}
}

func TestBigIntVeryLarge(t *testing.T) {
	// Test very large number that would overflow int64
	veryLarge := "12345678901234567890123456789012345678901234567890123456789012345678901234567890"
	os.Setenv("BIG_INT", veryLarge)
	defer os.Unsetenv("BIG_INT")

	cfg, err := Load(bigIntTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := new(big.Int)
	expected.SetString(veryLarge, 10)

	if cfg.BigInt.Cmp(expected) != 0 {
		t.Errorf("BigInt = %v; want %v", &cfg.BigInt, expected)
	}
}
