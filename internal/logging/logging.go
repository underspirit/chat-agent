package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Logger exposes the minimal functionality required by the ChatAgent skeleton.
type Logger interface {
	Info(msg string)
	Error(msg string)
	With(fields ...Field) Logger
}

// Field represents a structured logging key/value pair.
type Field struct {
	Key   string
	Value any
}

// stdLogger is a simple Logger implementation backed by the standard library log package.
type stdLogger struct {
	base  *log.Logger
	mu    sync.Mutex
	scope []Field
}

// NewStdLogger creates a logger writing to stdout.
func NewStdLogger() Logger {
	return NewStdLoggerWithWriter(os.Stdout)
}

// NewStdLoggerWithWriter creates a logger that writes to the provided io.Writer.
func NewStdLoggerWithWriter(w io.Writer) Logger {
	return &stdLogger{base: log.New(w, "chat-agent ", log.LstdFlags|log.Lmicroseconds)}
}

func (l *stdLogger) Info(msg string) {
	l.logWithLevel("INFO", msg)
}

func (l *stdLogger) Error(msg string) {
	l.logWithLevel("ERROR", msg)
}

func (l *stdLogger) With(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	next := &stdLogger{base: l.base, scope: append(append([]Field{}, l.scope...), fields...)}
	return next
}

func (l *stdLogger) logWithLevel(level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.scope) == 0 {
		l.base.Printf("[%s] %s", level, msg)
		return
	}

	l.base.Printf("[%s] %s %s", level, msg, renderFields(l.scope))
}

func renderFields(fields []Field) string {
	builder := strings.Builder{}
	builder.WriteString("[")
	for idx, f := range fields {
		if idx > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(fmt.Sprintf("%s=%v", f.Key, f.Value))
	}
	builder.WriteString("]")
	return builder.String()
}
