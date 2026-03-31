// Package logging provides a Zap-based structured logger for the CLI.
// By default the logger is silent (NopLogger). When --debug or SHOEHORN_DEBUG=1
// is set, it writes human-readable output to stderr at Debug level.
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a Zap logger for CLI use.
// When debug is true, a development-style logger writes to stderr.
// When debug is false, a no-op logger is returned (completely silent).
func New(debug bool) *zap.Logger {
	if !debug {
		return zap.NewNop()
	}

	// Human-readable output to stderr (stdout is for command results).
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderCfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(os.Stderr),
		zapcore.DebugLevel,
	)

	return zap.New(core)
}

// IsDebug returns true if SHOEHORN_DEBUG is set in the environment.
func IsDebug() bool {
	v := os.Getenv("SHOEHORN_DEBUG")
	return v == "1" || v == "true"
}
