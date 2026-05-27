package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Level represents a log verbosity level.
type Level int32

const (
	LevelDebug Level = iota
	LevelInfo
	LevelError
)

var (
	globalLevel atomic.Int32
	std         = log.New(os.Stderr, "", 0)
)

func init() {
	globalLevel.Store(int32(LevelInfo))
}

// SetLevel sets the global log level for all SDK loggers.
func SetLevel(level Level) {
	globalLevel.Store(int32(level))
}

// SetLevelFromString sets the global log level from a string ("DEBUG", "INFO", "ERROR").
func SetLevelFromString(s string) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		SetLevel(LevelDebug)
	case "ERROR":
		SetLevel(LevelError)
	default:
		SetLevel(LevelInfo)
	}
}

// Logger is a component-scoped structured logger.
type Logger struct {
	component string
}

// New returns a Logger scoped to the given component name.
func New(component string) *Logger {
	return &Logger{component: component}
}

func (l *Logger) enabled(lvl Level) bool {
	return lvl >= Level(globalLevel.Load())
}

func (l *Logger) output(levelStr, format string, args ...interface{}) {
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	std.Printf("%s %s [%s]: %s", now, levelStr, l.component, msg)
}

// Debug logs a DEBUG-level message (only when the global level is DEBUG).
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.enabled(LevelDebug) {
		l.output("DEBUG", format, args...)
	}
}

// Info logs an INFO-level message (when the global level is DEBUG or INFO).
func (l *Logger) Info(format string, args ...interface{}) {
	if l.enabled(LevelInfo) {
		l.output("INFO ", format, args...)
	}
}

// Error logs an ERROR-level message (always emitted).
func (l *Logger) Error(format string, args ...interface{}) {
	l.output("ERROR", format, args...)
}
