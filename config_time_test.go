package gonfig

import (
	"os"
	"testing"
	"time"
)

// timeTestConfig is used to test time-related type parsing
type timeTestConfig struct {
	Duration     time.Duration   `env:"DURATION"`
	TimeRFC3339  time.Time       `env:"TIME_RFC3339"`
	TimeUnix     time.Time       `env:"TIME_UNIX"`
	TimePtr      *time.Time      `env:"TIME_PTR"`
	DurationList []time.Duration `env:"DURATION_LIST"`
	TimeList     []time.Time     `env:"TIME_LIST"`
}

func TestTimeDuration(t *testing.T) {
	// Test time.Duration parsing
	os.Setenv("DURATION", "5m30s")
	defer os.Unsetenv("DURATION")

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := 5*time.Minute + 30*time.Second
	if cfg.Duration != expected {
		t.Errorf("Duration = %v; want %v", cfg.Duration, expected)
	}
}

func TestTimeDurationInvalid(t *testing.T) {
	// Test invalid duration
	os.Setenv("DURATION", "invalid")
	defer os.Unsetenv("DURATION")

	_, err := Load(timeTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid duration")
	}
}

func TestTimeRFC3339(t *testing.T) {
	// Test time.Time parsing with RFC3339 format
	timeStr := "2023-12-25T15:04:05Z"
	os.Setenv("TIME_RFC3339", timeStr)
	defer os.Unsetenv("TIME_RFC3339")

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t.Fatalf("Failed to parse expected time: %v", err)
	}

	if !cfg.TimeRFC3339.Equal(expected) {
		t.Errorf("TimeRFC3339 = %v; want %v", cfg.TimeRFC3339, expected)
	}
}

func TestTimeUnix(t *testing.T) {
	// Test time.Time parsing with Unix seconds
	unixTime := int64(1703516645) // 2023-12-25 15:04:05 UTC
	os.Setenv("TIME_UNIX", "1703516645")
	defer os.Unsetenv("TIME_UNIX")

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := time.Unix(unixTime, 0)
	if !cfg.TimeUnix.Equal(expected) {
		t.Errorf("TimeUnix = %v; want %v", cfg.TimeUnix, expected)
	}
}

func TestTimePtr(t *testing.T) {
	// Test *time.Time parsing
	timeStr := "2023-12-25T15:04:05Z"
	os.Setenv("TIME_PTR", timeStr)
	defer os.Unsetenv("TIME_PTR")

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.TimePtr == nil {
		t.Fatal("TimePtr should not be nil")
	}

	expected, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t.Fatalf("Failed to parse expected time: %v", err)
	}

	if !cfg.TimePtr.Equal(expected) {
		t.Errorf("TimePtr = %v; want %v", *cfg.TimePtr, expected)
	}
}

func TestTimeInvalid(t *testing.T) {
	// Test invalid time format
	os.Setenv("TIME_RFC3339", "invalid-time")
	defer os.Unsetenv("TIME_RFC3339")

	_, err := Load(timeTestConfig{})
	if err == nil {
		t.Error("Load should have failed with invalid time format")
	}
}

func TestDurationList(t *testing.T) {
	// Test []time.Duration parsing
	os.Setenv("DURATION_LIST", "1s,2m,3h")
	defer os.Unsetenv("DURATION_LIST")

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := []time.Duration{
		1 * time.Second,
		2 * time.Minute,
		3 * time.Hour,
	}

	if len(cfg.DurationList) != len(expected) {
		t.Fatalf("DurationList length = %d; want %d", len(cfg.DurationList), len(expected))
	}

	for i, duration := range cfg.DurationList {
		if duration != expected[i] {
			t.Errorf("DurationList[%d] = %v; want %v", i, duration, expected[i])
		}
	}
}

func TestTimeList(t *testing.T) {
	// Test []time.Time parsing
	os.Setenv("TIME_LIST", "2023-12-25T15:04:05Z,1703516645")
	defer os.Unsetenv("TIME_LIST")

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := []time.Time{
		time.Date(2023, 12, 25, 15, 4, 5, 0, time.UTC),
		time.Unix(1703516645, 0),
	}

	if len(cfg.TimeList) != len(expected) {
		t.Fatalf("TimeList length = %d; want %d", len(cfg.TimeList), len(expected))
	}

	for i, timeVal := range cfg.TimeList {
		if !timeVal.Equal(expected[i]) {
			t.Errorf("TimeList[%d] = %v; want %v", i, timeVal, expected[i])
		}
	}
}

func TestTimeDefaults(t *testing.T) {
	// Test default values for time types
	type timeDefaultConfig struct {
		Duration time.Duration `env:"MISSING_DURATION" default:"10s"`
		Time     time.Time     `env:"MISSING_TIME" default:"2023-01-01T00:00:00Z"`
	}

	cfg, err := Load(timeDefaultConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expectedDuration := 10 * time.Second
	if cfg.Duration != expectedDuration {
		t.Errorf("Duration = %v; want %v", cfg.Duration, expectedDuration)
	}

	expectedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if !cfg.Time.Equal(expectedTime) {
		t.Errorf("Time = %v; want %v", cfg.Time, expectedTime)
	}
}

func TestTimeRequired(t *testing.T) {
	// Test required time fields
	type timeRequiredConfig struct {
		Duration time.Duration `env:"REQUIRED_DURATION" required:"true"`
	}

	_, err := Load(timeRequiredConfig{})
	if err == nil {
		t.Error("Load should have failed with missing required duration")
	}
}

func TestTimeEmptyList(t *testing.T) {
	// Test empty time lists
	os.Setenv("DURATION_LIST", "")
	os.Setenv("TIME_LIST", "")
	defer func() {
		os.Unsetenv("DURATION_LIST")
		os.Unsetenv("TIME_LIST")
	}()

	cfg, err := Load(timeTestConfig{})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.DurationList) != 0 {
		t.Errorf("DurationList length = %d; want 0", len(cfg.DurationList))
	}

	if len(cfg.TimeList) != 0 {
		t.Errorf("TimeList length = %d; want 0", len(cfg.TimeList))
	}
}
