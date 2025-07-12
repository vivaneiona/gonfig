package getconfig

import (
	"os"
	"testing"

	"github.com/shopspring/decimal"
)

// decimalTestConfig is used to test decimal.Decimal parsing
type decimalTestConfig struct {
	Decimal        decimal.Decimal    `env:"DECIMAL"`
	DecimalPtr     *decimal.Decimal   `env:"DECIMAL_PTR"`
	DecimalList    []decimal.Decimal  `env:"DECIMAL_LIST"`
	DecimalPtrList []*decimal.Decimal `env:"DECIMAL_PTR_LIST"`
	Price          decimal.Decimal    `env:"PRICE" default:"19.99"`
	Commission     decimal.Decimal    `env:"COMMISSION" default:"0.001"`
	Zero           decimal.Decimal    `env:"ZERO" default:"0"`
}

func TestDecimalBasic(t *testing.T) {
	// Test basic decimal parsing
	os.Setenv("DECIMAL", "123.456")
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, _ := decimal.NewFromString("123.456")
	if !cfg.Decimal.Equal(expected) {
		t.Errorf("Decimal = %v; want %v", cfg.Decimal, expected)
	}
}

func TestDecimalPtr(t *testing.T) {
	// Test *decimal.Decimal parsing
	os.Setenv("DECIMAL_PTR", "987.654321")
	defer os.Unsetenv("DECIMAL_PTR")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DecimalPtr == nil {
		t.Fatal("DecimalPtr should not be nil")
	}

	expected, _ := decimal.NewFromString("987.654321")
	if !cfg.DecimalPtr.Equal(expected) {
		t.Errorf("DecimalPtr = %v; want %v", *cfg.DecimalPtr, expected)
	}
}

func TestDecimalNegative(t *testing.T) {
	// Test negative decimal
	os.Setenv("DECIMAL", "-123.456")
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, _ := decimal.NewFromString("-123.456")
	if !cfg.Decimal.Equal(expected) {
		t.Errorf("Decimal = %v; want %v", cfg.Decimal, expected)
	}
}

func TestDecimalZero(t *testing.T) {
	// Test zero decimal
	os.Setenv("DECIMAL", "0")
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := decimal.Zero
	if !cfg.Decimal.Equal(expected) {
		t.Errorf("Decimal = %v; want %v", cfg.Decimal, expected)
	}
}

func TestDecimalInteger(t *testing.T) {
	// Test integer input (should work as decimal)
	os.Setenv("DECIMAL", "42")
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := decimal.NewFromInt(42)
	if !cfg.Decimal.Equal(expected) {
		t.Errorf("Decimal = %v; want %v", cfg.Decimal, expected)
	}
}

func TestDecimalScientificNotation(t *testing.T) {
	// Test scientific notation
	os.Setenv("DECIMAL", "1.23e-4")
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, _ := decimal.NewFromString("0.000123")
	if !cfg.Decimal.Equal(expected) {
		t.Errorf("Decimal = %v; want %v", cfg.Decimal, expected)
	}
}

func TestDecimalVeryPrecise(t *testing.T) {
	// Test very precise decimal (more precision than float64)
	preciseValue := "123.123456789012345678901234567890"
	os.Setenv("DECIMAL", preciseValue)
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, _ := decimal.NewFromString(preciseValue)
	if !cfg.Decimal.Equal(expected) {
		t.Errorf("Decimal = %v; want %v", cfg.Decimal, expected)
	}
}

func TestDecimalInvalid(t *testing.T) {
	// Test invalid decimal
	os.Setenv("DECIMAL", "not-a-number")
	defer os.Unsetenv("DECIMAL")

	// Load into config struct and capture returned cfg and error
	_, err := Load(decimalTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid decimal")
	}
}

func TestDecimalList(t *testing.T) {
	// Test []decimal.Decimal parsing
	os.Setenv("DECIMAL_LIST", "1.23,4.56,-7.89,0.001")
	defer os.Unsetenv("DECIMAL_LIST")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"1.23", "4.56", "-7.89", "0.001"}
	if len(cfg.DecimalList) != len(expectedValues) {
		t.Fatalf("DecimalList length = %d; want %d", len(cfg.DecimalList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		expected, _ := decimal.NewFromString(expectedStr)
		if !cfg.DecimalList[i].Equal(expected) {
			t.Errorf("DecimalList[%d] = %v; want %v", i, cfg.DecimalList[i], expected)
		}
	}
}

func TestDecimalPtrList(t *testing.T) {
	// Test []*decimal.Decimal parsing
	os.Setenv("DECIMAL_PTR_LIST", "10.5,20.25")
	defer os.Unsetenv("DECIMAL_PTR_LIST")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"10.5", "20.25"}
	if len(cfg.DecimalPtrList) != len(expectedValues) {
		t.Fatalf("DecimalPtrList length = %d; want %d", len(cfg.DecimalPtrList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		if cfg.DecimalPtrList[i] == nil {
			t.Fatalf("DecimalPtrList[%d] should not be nil", i)
		}

		expected, _ := decimal.NewFromString(expectedStr)
		if !cfg.DecimalPtrList[i].Equal(expected) {
			t.Errorf("DecimalPtrList[%d] = %v; want %v", i, *cfg.DecimalPtrList[i], expected)
		}
	}
}

func TestDecimalDefaults(t *testing.T) {
	// Test default values for decimal
	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check price default
	expectedPrice, _ := decimal.NewFromString("19.99")
	if !cfg.Price.Equal(expectedPrice) {
		t.Errorf("Price = %v; want %v", cfg.Price, expectedPrice)
	}

	// Check commission default
	expectedCommission, _ := decimal.NewFromString("0.001")
	if !cfg.Commission.Equal(expectedCommission) {
		t.Errorf("Commission = %v; want %v", cfg.Commission, expectedCommission)
	}

	// Check zero default
	expectedZero := decimal.Zero
	if !cfg.Zero.Equal(expectedZero) {
		t.Errorf("Zero = %v; want %v", cfg.Zero, expectedZero)
	}
}

func TestDecimalRequired(t *testing.T) {
	// Test required decimal field
	type decimalRequiredConfig struct {
		Decimal decimal.Decimal `env:"REQUIRED_DECIMAL" required:"true"`
	}

	// Load into config struct and capture returned cfg and error
	_, err := Load(decimalRequiredConfig{})
	if err == nil {
		t.Error("Load should have failed with missing required decimal")
	}
}

func TestDecimalEmptyList(t *testing.T) {
	// Test empty decimal list
	os.Setenv("DECIMAL_LIST", "")
	defer os.Unsetenv("DECIMAL_LIST")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.DecimalList) != 0 {
		t.Errorf("DecimalList length = %d; want 0", len(cfg.DecimalList))
	}
}

func TestDecimalSpacesInList(t *testing.T) {
	// Test decimal list with spaces
	os.Setenv("DECIMAL_LIST", " 1.23 , 4.56 , 7.89 ")
	defer os.Unsetenv("DECIMAL_LIST")

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(decimalTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"1.23", "4.56", "7.89"}
	if len(cfg.DecimalList) != len(expectedValues) {
		t.Fatalf("DecimalList length = %d; want %d", len(cfg.DecimalList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		expected, _ := decimal.NewFromString(expectedStr)
		if !cfg.DecimalList[i].Equal(expected) {
			t.Errorf("DecimalList[%d] = %v; want %v", i, cfg.DecimalList[i], expected)
		}
	}
}

func TestDecimalMoneyExample(t *testing.T) {
	// Test practical money example
	type moneyConfig struct {
		Price    decimal.Decimal `env:"ITEM_PRICE"`
		Tax      decimal.Decimal `env:"TAX_RATE"`
		Discount decimal.Decimal `env:"DISCOUNT"`
	}

	os.Setenv("ITEM_PRICE", "99.99")
	os.Setenv("TAX_RATE", "0.0825") // 8.25% tax
	os.Setenv("DISCOUNT", "10.00")
	defer func() {
		os.Unsetenv("ITEM_PRICE")
		os.Unsetenv("TAX_RATE")
		os.Unsetenv("DISCOUNT")
	}()

	// Load into config struct and capture returned cfg and error
	cfg, err := Load(moneyConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify exact decimal arithmetic
	expectedPrice, _ := decimal.NewFromString("99.99")
	expectedTax, _ := decimal.NewFromString("0.0825")
	expectedDiscount, _ := decimal.NewFromString("10.00")

	if !cfg.Price.Equal(expectedPrice) {
		t.Errorf("Price = %v; want %v", cfg.Price, expectedPrice)
	}

	if !cfg.Tax.Equal(expectedTax) {
		t.Errorf("Tax = %v; want %v", cfg.Tax, expectedTax)
	}

	if !cfg.Discount.Equal(expectedDiscount) {
		t.Errorf("Discount = %v; want %v", cfg.Discount, expectedDiscount)
	}

	// Calculate final price: (price - discount) * (1 + tax)
	finalPrice := cfg.Price.Sub(cfg.Discount).Mul(decimal.NewFromInt(1).Add(cfg.Tax))
	expectedFinal, _ := decimal.NewFromString("97.414175") // (99.99 - 10.00) * 1.0825 = 89.99 * 1.0825

	if !finalPrice.Equal(expectedFinal) {
		t.Errorf("Final price = %v; want %v", finalPrice, expectedFinal)
	}
}
