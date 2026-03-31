package logging

import (
	"testing"
)

func TestNew_DebugFalse_ReturnsNopLogger(t *testing.T) {
	l := New(false)
	if l == nil {
		t.Fatal("New(false) returned nil")
	}
	// NopLogger should not panic when logging
	l.Debug("test message")
	l.Info("test message")
	l.Warn("test message")
}

func TestNew_DebugTrue_ReturnsActiveLogger(t *testing.T) {
	l := New(true)
	if l == nil {
		t.Fatal("New(true) returned nil")
	}
	// Active logger should write to stderr without panicking
	l.Debug("test debug message")
	l.Info("test info message")
}

func TestIsDebug_NotSet(t *testing.T) {
	t.Setenv("SHOEHORN_DEBUG", "")
	if IsDebug() {
		t.Error("IsDebug() = true with empty env")
	}
}

func TestIsDebug_SetTo1(t *testing.T) {
	t.Setenv("SHOEHORN_DEBUG", "1")
	if !IsDebug() {
		t.Error("IsDebug() = false with SHOEHORN_DEBUG=1")
	}
}

func TestIsDebug_SetToTrue(t *testing.T) {
	t.Setenv("SHOEHORN_DEBUG", "true")
	if !IsDebug() {
		t.Error("IsDebug() = false with SHOEHORN_DEBUG=true")
	}
}

func TestIsDebug_SetToOtherValue(t *testing.T) {
	t.Setenv("SHOEHORN_DEBUG", "yes")
	if IsDebug() {
		t.Error("IsDebug() = true with SHOEHORN_DEBUG=yes (only '1' and 'true' accepted)")
	}
}
