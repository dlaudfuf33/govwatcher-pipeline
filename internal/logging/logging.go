package logging

import (
	"fmt"
	"log"
	"strings"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var currentLevel = DEBUG // 기본값은 DEBUG

func SetLevel(level LogLevel) {
	currentLevel = level
}

func parsePrefix(level LogLevel) string {
	switch level {
	case DEBUG:
		return "🐛 [DEBUG]:"
	case INFO:
		return "ℹ️ [INFO]:"
	case WARN:
		return "⚠️ [WARN]:"
	case ERROR:
		return "❌ [ERROR]:"
	default:
		return "[LOG]"
	}
}

func logf(level LogLevel, format string, v ...any) {
	if level >= currentLevel {
		prefix := parsePrefix(level)
		log.Printf("%s %s", prefix, fmt.Sprintf(format, v...))
	}
}

func Debugf(format string, v ...any) {
	logf(DEBUG, format, v...)
}

func Infof(format string, v ...any) {
	logf(INFO, format, v...)
}

func Warnf(format string, v ...any) {
	logf(WARN, format, v...)
}

func Errorf(format string, v ...any) {
	logf(ERROR, format, v...)
}

// 문자열 → LogLevel
func ParseLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}
