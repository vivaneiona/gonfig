package gonfig

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

// quantityTestConfig demonstrates Kubernetes resource.Quantity parsing
type quantityTestConfig struct {
	// CPU and memory limits using Kubernetes resource units
	CPULimit  resource.Quantity `env:"CPU_LIMIT"`
	MemLimit  resource.Quantity `env:"MEM_LIMIT"`
	DiskLimit resource.Quantity `env:"DISK_LIMIT"`
	Bandwidth resource.Quantity `env:"BANDWIDTH"`

	// Pointer versions
	CPUPtr *resource.Quantity `env:"CPU_PTR"`
	MemPtr *resource.Quantity `env:"MEM_PTR"`

	// Lists of quantities
	CPUList []resource.Quantity `env:"CPU_LIST"`
	MemList []resource.Quantity `env:"MEM_LIST"`

	// Default values demonstrating different unit formats
	DefaultCPU  resource.Quantity `env:"DEFAULT_CPU" default:"500m"`
	DefaultMem  resource.Quantity `env:"DEFAULT_MEM" default:"1Gi"`
	DefaultDisk resource.Quantity `env:"DEFAULT_DISK" default:"10G"`
}

// cfg is the shared config instance for quantity tests
var cfg quantityTestConfig

func TestQuantityCPUMillicores(t *testing.T) {
	// Test CPU in millicores
	os.Setenv("CPU_LIMIT", "250m")
	defer os.Unsetenv("CPU_LIMIT")
	var cfg quantityTestConfig
	var err error
	cfg, err = Load(quantityTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := resource.MustParse("250m")
	if !cfg.CPULimit.Equal(expected) {
		t.Errorf("CPULimit = %v; want %v", cfg.CPULimit, expected)
	}

	// Verify it's actually 0.25 cores
	milliValue := cfg.CPULimit.MilliValue()
	if milliValue != 250 {
		t.Errorf("CPULimit MilliValue = %d; want 250", milliValue)
	}
}

func TestQuantityMemoryBinary(t *testing.T) {
	// Test memory in binary units (Gi = 1024^3 bytes)
	os.Setenv("MEM_LIMIT", "1.5Gi")
	defer os.Unsetenv("MEM_LIMIT")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := resource.MustParse("1.5Gi")
	if !cfg.MemLimit.Equal(expected) {
		t.Errorf("MemLimit = %v; want %v", cfg.MemLimit, expected)
	}

	// Verify it's the correct number of bytes (1.5 * 1024^3)
	bytes := cfg.MemLimit.Value()
	expectedBytes := int64(1.5 * 1024 * 1024 * 1024)
	if bytes != expectedBytes {
		t.Errorf("MemLimit bytes = %d; want %d", bytes, expectedBytes)
	}
}

func TestQuantityMemoryDecimal(t *testing.T) {
	// Test memory in decimal units (G = 10^9 bytes in Kubernetes)
	os.Setenv("MEM_LIMIT", "2G")
	defer os.Unsetenv("MEM_LIMIT")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := resource.MustParse("2G")
	if !cfg.MemLimit.Equal(expected) {
		t.Errorf("MemLimit = %v; want %v", cfg.MemLimit, expected)
	}

	// Verify it's 2 billion bytes
	bytes := cfg.MemLimit.Value()
	if bytes != 2_000_000_000 {
		t.Errorf("MemLimit bytes = %d; want 2000000000", bytes)
	}
}

func TestQuantityDiskStorage(t *testing.T) {
	// Test disk storage in various units
	os.Setenv("DISK_LIMIT", "500Mi")
	defer os.Unsetenv("DISK_LIMIT")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := resource.MustParse("500Mi")
	if !cfg.DiskLimit.Equal(expected) {
		t.Errorf("DiskLimit = %v; want %v", cfg.DiskLimit, expected)
	}

	// Verify it's 500 * 1024^2 bytes
	bytes := cfg.DiskLimit.Value()
	expectedBytes := int64(500 * 1024 * 1024)
	if bytes != expectedBytes {
		t.Errorf("DiskLimit bytes = %d; want %d", bytes, expectedBytes)
	}
}

func TestQuantityBandwidth(t *testing.T) {
	// Test large quantities (bandwidth in bytes)
	os.Setenv("BANDWIDTH", "100Mi")
	defer os.Unsetenv("BANDWIDTH")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := resource.MustParse("100Mi")
	if !cfg.Bandwidth.Equal(expected) {
		t.Errorf("Bandwidth = %v; want %v", cfg.Bandwidth, expected)
	}
}

func TestQuantityZero(t *testing.T) {
	// Test zero quantity
	os.Setenv("CPU_LIMIT", "0")
	defer os.Unsetenv("CPU_LIMIT")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := resource.MustParse("0")
	if !cfg.CPULimit.Equal(expected) {
		t.Errorf("CPULimit = %v; want %v", cfg.CPULimit, expected)
	}

	if !cfg.CPULimit.IsZero() {
		t.Error("CPULimit should be zero")
	}
}

func TestQuantityPointer(t *testing.T) {
	// Test *resource.Quantity
	os.Setenv("CPU_PTR", "2")
	defer os.Unsetenv("CPU_PTR")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.CPUPtr == nil {
		t.Fatal("CPUPtr should not be nil")
	}

	expected := resource.MustParse("2")
	if !cfg.CPUPtr.Equal(expected) {
		t.Errorf("CPUPtr = %v; want %v", *cfg.CPUPtr, expected)
	}
}

func TestQuantityList(t *testing.T) {
	// Test []resource.Quantity
	os.Setenv("CPU_LIST", "100m,250m,500m,1")
	defer os.Unsetenv("CPU_LIST")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"100m", "250m", "500m", "1"}
	if len(cfg.CPUList) != len(expectedValues) {
		t.Fatalf("CPUList length = %d; want %d", len(cfg.CPUList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		expected := resource.MustParse(expectedStr)
		if !cfg.CPUList[i].Equal(expected) {
			t.Errorf("CPUList[%d] = %v; want %v", i, cfg.CPUList[i], expected)
		}
	}
}

func TestQuantityMixedUnits(t *testing.T) {
	// Test mixed binary and decimal units
	os.Setenv("MEM_LIST", "1Gi,2G,512Mi,1500G")
	defer os.Unsetenv("MEM_LIST")

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedValues := []string{"1Gi", "2G", "512Mi", "1500G"}
	if len(cfg.MemList) != len(expectedValues) {
		t.Fatalf("MemList length = %d; want %d", len(cfg.MemList), len(expectedValues))
	}

	for i, expectedStr := range expectedValues {
		expected := resource.MustParse(expectedStr)
		if !cfg.MemList[i].Equal(expected) {
			t.Errorf("MemList[%d] = %v; want %v", i, cfg.MemList[i], expected)
		}
	}
}

func TestQuantityDefaults(t *testing.T) {
	// Test default values
	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check CPU default (500m = 0.5 cores)
	expectedCPU := resource.MustParse("500m")
	if !cfg.DefaultCPU.Equal(expectedCPU) {
		t.Errorf("DefaultCPU = %v; want %v", cfg.DefaultCPU, expectedCPU)
	}

	// Check memory default (1Gi)
	expectedMem := resource.MustParse("1Gi")
	if !cfg.DefaultMem.Equal(expectedMem) {
		t.Errorf("DefaultMem = %v; want %v", cfg.DefaultMem, expectedMem)
	}

	// Check disk default (10G)
	expectedDisk := resource.MustParse("10G")
	if !cfg.DefaultDisk.Equal(expectedDisk) {
		t.Errorf("DefaultDisk = %v; want %v", cfg.DefaultDisk, expectedDisk)
	}
}

func TestQuantityInvalid(t *testing.T) {
	// Test invalid quantity format
	os.Setenv("CPU_LIMIT", "invalid-quantity")
	defer os.Unsetenv("CPU_LIMIT")

	_, err := Load(cfg)
	if err == nil {
		t.Error("Load should have failed with invalid quantity")
	}
}

func TestQuantityRequired(t *testing.T) {
	// Test required quantity field
	type quantityRequiredConfig struct {
		CPU resource.Quantity `env:"REQUIRED_CPU" required:"true"`
	}

	_, err := Load(quantityRequiredConfig{})
	if err == nil {
		t.Error("Load should have failed with missing required quantity")
	}
}

func TestQuantityComparison(t *testing.T) {
	// Test quantity comparison capabilities
	os.Setenv("CPU_LIMIT", "500m")
	os.Setenv("MEM_LIMIT", "2Gi")
	defer func() {
		os.Unsetenv("CPU_LIMIT")
		os.Unsetenv("MEM_LIMIT")
	}()

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test CPU comparison
	halfCore := resource.MustParse("500m")
	oneCore := resource.MustParse("1")

	if cfg.CPULimit.Cmp(halfCore) != 0 {
		t.Error("CPU should equal 500m")
	}

	if cfg.CPULimit.Cmp(oneCore) >= 0 {
		t.Error("500m should be less than 1 core")
	}

	// Test memory comparison
	twoGi := resource.MustParse("2Gi")
	fourGi := resource.MustParse("4Gi")

	if cfg.MemLimit.Cmp(twoGi) != 0 {
		t.Error("Memory should equal 2Gi")
	}

	if cfg.MemLimit.Cmp(fourGi) >= 0 {
		t.Error("2Gi should be less than 4Gi")
	}
}

func TestQuantityArithmetic(t *testing.T) {
	// Test quantity arithmetic operations
	os.Setenv("CPU_LIMIT", "250m")
	os.Setenv("MEM_LIMIT", "1Gi")
	defer func() {
		os.Unsetenv("CPU_LIMIT")
		os.Unsetenv("MEM_LIMIT")
	}()

	var err error
	cfg, err = Load(cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test CPU arithmetic
	quarter := cfg.CPULimit.DeepCopy()
	quarter.Add(resource.MustParse("250m"))
	halfCore := resource.MustParse("500m")

	if quarter.Cmp(halfCore) != 0 {
		t.Errorf("250m + 250m should equal 500m, got %v", quarter)
	}

	// Test memory arithmetic
	oneGi := cfg.MemLimit.DeepCopy()
	oneGi.Add(resource.MustParse("1Gi"))
	twoGi := resource.MustParse("2Gi")

	if oneGi.Cmp(twoGi) != 0 {
		t.Errorf("1Gi + 1Gi should equal 2Gi, got %v", oneGi)
	}
}
