package logger

import (
	"context"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestLogger_BasicOutput(t *testing.T) {
	var buf strings.Builder
	l := NewLogger(
		WithWriter(&buf),
		WithAppName("UnitTest"),
		WithColor(false),
	)

	// Testa log básico
	l.Info("hello world")

	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("Output does not contain message: got %q", out)
	}

	if !strings.Contains(out, "[INFO]") {
		t.Errorf("Output does not contain level: got %q", out)
	}

	if !strings.Contains(out, "[UnitTest]") {
		t.Errorf("Output does not contain app name: got %q", out)
	}

	buf.Reset()

	// Testa log com extras
	l.Warn("warning message", "foo", "bar", "x", 42)

	out = buf.String()

	if !strings.Contains(out, "warning message") ||
		!strings.Contains(out, "foo=bar") ||
		!strings.Contains(out, "x=42") ||
		!strings.Contains(out, "[WARN]") {
		t.Errorf("Output does not contain all expected values: got %q", out)
	}

	buf.Reset()

	// Testa log com contexto (simples, sem trace)
	l.ErrorCtx(context.Background(), "error ctx")

	out = buf.String()
	if !strings.Contains(out, "error ctx") || !strings.Contains(out, "[ERROR]") {
		t.Errorf("Output does not contain context error: got %q", out)
	}

	buf.Reset()
}

func TestLogger_CustomFormat(t *testing.T) {
	var buf strings.Builder
	l := NewLogger(
		WithWriter(&buf),
		WithFormat("{level} - {message}"),
		WithColor(false),
	)
	l.Info("custom format", "a", 1)
	out := buf.String()
	if !strings.HasPrefix(out, "INFO - custom format") {
		t.Errorf("Custom format failed: got %q", out)
	}
}

func TestLogger_ColorOutput(t *testing.T) {
	var buf strings.Builder
	l := NewLogger(WithWriter(&buf), WithColor(true))

	// Cores são códigos ANSI, então para facilitar só verifica se tem o código no output
	l.Debug("debugging!")
	out := buf.String()
	if !strings.Contains(out, "\033[36mDEBUG\033[0m") {
		t.Errorf("Output should contain cyan color for DEBUG: got %q", out)
	}
}

// Teste para argumento extra ímpar
func TestLogger_ExtraArgsOdd(t *testing.T) {
	var buf strings.Builder
	l := NewLogger(WithWriter(&buf), WithColor(false))
	l.Info("test", "onlyKey")
	out := buf.String()
	if !strings.Contains(out, "test") {
		t.Errorf("Should handle odd extra args gracefully: got %q", out)
	}
}

func TestLogger_WithOtelSpan(t *testing.T) {
	var buf strings.Builder

	l := NewLogger(
		WithWriter(&buf),
		WithFormat("{message} {trace_id} {span_id} {extra}"),
		WithColor(false),
	)

	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-logger")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	l.InfoCtx(ctx, "log with trace", "foo", "bar")
	out := buf.String()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	if traceID == "" || spanID == "" {
		t.Fatal("traceID or spanID should not be empty")
	}
	if !strings.Contains(out, "trace_id="+traceID) {
		t.Errorf("Log should contain trace_id, got: %q", out)
	}
	if !strings.Contains(out, "span_id="+spanID) {
		t.Errorf("Log should contain span_id, got: %q", out)
	}
	if !strings.Contains(out, "log with trace") || !strings.Contains(out, "foo=bar") {
		t.Errorf("Log missing expected content: %q", out)
	}
}
