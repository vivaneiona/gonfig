package getconfig

import (
	"log/slog"
	"os"
	"testing"
)

// slogTestConfig is used to test slog.Level parsing
type slogTestConfig struct {
	Level      slog.Level   `env:"LOG_LEVEL"`
	LevelPtr   *slog.Level  `env:"LOG_LEVEL_PTR"`
	LevelList  []slog.Level `env:"LOG_LEVEL_LIST"`
	DebugLevel slog.Level   `env:"DEBUG_LEVEL" default:"debug"`
	InfoLevel  slog.Level   `env:"INFO_LEVEL" default:"info"`
	WarnLevel  slog.Level   `env:"WARN_LEVEL" default:"warn"`
	ErrorLevel slog.Level   `env:"ERROR_LEVEL" default:"error"`
}

func TestSlogLevelDebug(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Level != slog.LevelDebug {
		t.Errorf("Level = %v; want %v", cfg.Level, slog.LevelDebug)
	}
}

func TestSlogLevelInfo(t *testing.T) {
	os.Setenv("LOG_LEVEL", "info")
	defer os.Unsetenv("LOG_LEVEL")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Level != slog.LevelInfo {
		t.Errorf("Level = %v; want %v", cfg.Level, slog.LevelInfo)
	}
}

func TestSlogLevelWarn(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Level != slog.LevelWarn {
		t.Errorf("Level = %v; want %v", cfg.Level, slog.LevelWarn)
	}
}

func TestSlogLevelWarning(t *testing.T) {
	// Test "warning" as alias for "warn"
	os.Setenv("LOG_LEVEL", "warning")
	defer os.Unsetenv("LOG_LEVEL")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Level != slog.LevelWarn {
		t.Errorf("Level = %v; want %v", cfg.Level, slog.LevelWarn)
	}
}

func TestSlogLevelError(t *testing.T) {
	os.Setenv("LOG_LEVEL", "error")
	defer os.Unsetenv("LOG_LEVEL")

	if cfg, err := Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	} else {
		if cfg.Level != slog.LevelError {
			t.Errorf("Level = %v; want %v", cfg.Level, slog.LevelError)
		}
	}
}

func TestSlogLevelCaseInsensitive(t *testing.T) {
	// Test case insensitive parsing
	testCases := []string{"DEBUG", "Info", "WARN", "Error"}
	expected := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}

	for i, levelStr := range testCases {
		os.Setenv("LOG_LEVEL", levelStr)
		var cfg slogTestConfig
		var err error
		if cfg, err = Load(slogTestConfig{}); err != nil {
			t.Fatalf("Load failed for %s: %v", levelStr, err)
		}

		if cfg.Level != expected[i] {
			t.Errorf("Level for %s = %v; want %v", levelStr, cfg.Level, expected[i])
		}
		os.Unsetenv("LOG_LEVEL")
	}
}

func TestSlogLevelInteger(t *testing.T) {
	// Test integer level parsing
	os.Setenv("LOG_LEVEL", "12")
	defer os.Unsetenv("LOG_LEVEL")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Level != slog.Level(12) {
		t.Errorf("Level = %v; want %v", cfg.Level, slog.Level(12))
	}
}

func TestSlogLevelNegativeInteger(t *testing.T) {
	// Test negative integer level parsing
	os.Setenv("LOG_LEVEL", "-4")
	defer os.Unsetenv("LOG_LEVEL")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Level != slog.Level(-4) {
		t.Errorf("Level = %v; want %v", cfg.Level, slog.Level(-4))
	}
}

func TestSlogLevelPtr(t *testing.T) {
	// Test *slog.Level parsing
	os.Setenv("LOG_LEVEL_PTR", "info")
	defer os.Unsetenv("LOG_LEVEL_PTR")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LevelPtr == nil {
		t.Fatal("LevelPtr should not be nil")
	}

	if *cfg.LevelPtr != slog.LevelInfo {
		t.Errorf("LevelPtr = %v; want %v", *cfg.LevelPtr, slog.LevelInfo)
	}
}

func TestSlogLevelInvalid(t *testing.T) {
	// Test invalid slog level
	os.Setenv("LOG_LEVEL", "invalid")
	defer os.Unsetenv("LOG_LEVEL")
	var err error
	if _, err = Load(slogTestConfig{}); err == nil {
		t.Error("Load should have failed with invalid slog level")
	}
}

func TestSlogLevelList(t *testing.T) {
	// Test []slog.Level parsing
	os.Setenv("LOG_LEVEL_LIST", "debug,info,warn,error")
	defer os.Unsetenv("LOG_LEVEL_LIST")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	if len(cfg.LevelList) != len(expected) {
		t.Fatalf("LevelList length = %d; want %d", len(cfg.LevelList), len(expected))
	}

	for i, level := range cfg.LevelList {
		if level != expected[i] {
			t.Errorf("LevelList[%d] = %v; want %v", i, level, expected[i])
		}
	}
}

func TestSlogLevelDefaults(t *testing.T) {
	// Test default values for slog levels
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DebugLevel != slog.LevelDebug {
		t.Errorf("DebugLevel = %v; want %v", cfg.DebugLevel, slog.LevelDebug)
	}

	if cfg.InfoLevel != slog.LevelInfo {
		t.Errorf("InfoLevel = %v; want %v", cfg.InfoLevel, slog.LevelInfo)
	}

	if cfg.WarnLevel != slog.LevelWarn {
		t.Errorf("WarnLevel = %v; want %v", cfg.WarnLevel, slog.LevelWarn)
	}

	if cfg.ErrorLevel != slog.LevelError {
		t.Errorf("ErrorLevel = %v; want %v", cfg.ErrorLevel, slog.LevelError)
	}
}

func TestSlogLevelRequired(t *testing.T) {
	// Test required slog level field
	type slogRequiredConfig struct {
		Level slog.Level `env:"REQUIRED_LOG_LEVEL" required:"true"`
	}
	var err error
	if _, err = Load(slogRequiredConfig{}); err == nil {
		t.Error("Load should have failed with missing required slog level")
	}
}

func TestSlogLevelEmptyList(t *testing.T) {
	// Test empty slog level list
	os.Setenv("LOG_LEVEL_LIST", "")
	defer os.Unsetenv("LOG_LEVEL_LIST")
	var cfg slogTestConfig
	var err error
	if cfg, err = Load(slogTestConfig{}); err != nil {
		t.Fatalf("Load failed: %v", err)
	} else {
		if len(cfg.LevelList) != 0 {
			t.Errorf("LevelList length = %d; want 0", len(cfg.LevelList))
		}
	}
}
