package wslogger

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/natefinch/lumberjack"
	"go.opentelemetry.io/otel/attribute"
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

func TestLogger_JSONOutput(t *testing.T) {
	var buf strings.Builder
	l := NewLogger(
		WithWriter(&buf),
		WithJSON(true),
	)

	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-logger")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	l.InfoCtx(ctx, "log as json", "foo", "bar")

	var logRecord map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &logRecord); err != nil {
		t.Fatalf("Failed to unmarshal log json: %v", err)
	}

	if logRecord["message"] != "log as json" {
		t.Errorf("JSON missing message: %v", logRecord)
	}
	if logRecord["trace_id"] != span.SpanContext().TraceID().String() {
		t.Errorf("JSON missing trace_id: %v", logRecord)
	}
	if logRecord["span_id"] != span.SpanContext().SpanID().String() {
		t.Errorf("JSON missing span_id: %v", logRecord)
	}
	extra, ok := logRecord["extra"].(map[string]interface{})
	if !ok || extra["foo"] != "bar" {
		t.Errorf("JSON missing extra fields: %v", logRecord)
	}
}

func TestLogger_JSON_WithSpanAttributes(t *testing.T) {
	var buf strings.Builder
	logger := NewLogger(
		WithWriter(&buf),
		WithJSON(true),
		WithSpanAttributes(true),
	)

	// Cria contexto com span e atributos
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("logger-test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	span.SetAttributes(
		attribute.String("user_id", "1234"),
		attribute.String("custom_role", "superadmin"),
	)
	defer span.End()

	// Inclui extra
	logger.InfoCtx(ctx, "test message", "foo", "bar")

	// Parse do JSON
	var record map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &record); err != nil {
		t.Fatalf("failed to unmarshal log: %v\nlog line: %q", err, buf.String())
	}

	// Valida message
	if record["message"] != "test message" {
		t.Errorf("expected message %q, got %q", "test message", record["message"])
	}

	// Valida trace_id/span_id
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	if record["trace_id"] != traceID {
		t.Errorf("expected trace_id %q, got %q", traceID, record["trace_id"])
	}
	if record["span_id"] != spanID {
		t.Errorf("expected span_id %q, got %q", spanID, record["span_id"])
	}

	// Valida extras
	extra, ok := record["extra"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing or invalid extra: %v", record["extra"])
	}
	if extra["foo"] != "bar" {
		t.Errorf("expected foo=bar in extra, got %v", extra["foo"])
	}
	if extra["user_id"] != "1234" {
		t.Errorf("expected user_id=1234 in extra, got %v", extra["user_id"])
	}
	if extra["custom_role"] != "superadmin" {
		t.Errorf("expected custom_role=superadmin in extra, got %v", extra["custom_role"])
	}
}

func TestLogger_LogRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "testlog.log")

	log := NewLogger(
		WithJSON(true),
		WithRotatingFile(logFile, 1, 3, 1, false),
	)

	// 8KB por linha, 200 linhas = 1.6MB
	bigMsg := strings.Repeat("Y", 8192)
	lines := 200
	for i := 0; i < lines; i++ {
		log.Info(bigMsg, "i", strconv.Itoa(i))
	}

	// Força rotação explícita
	if lj, ok := log.writer.(*lumberjack.Logger); ok {
		_ = lj.Rotate()
	}

	time.Sleep(500 * time.Millisecond)

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read tmp log dir: %v", err)
	}

	for _, f := range files {
		info, _ := f.Info()
		t.Logf("Arquivo: %s (%d bytes)", f.Name(), info.Size())
	}

	foundMain := false
	foundBackup := false

	for _, f := range files {
		if f.Name() == "testlog.log" {
			foundMain = true
		}
		if strings.HasPrefix(f.Name(), "testlog-") && strings.HasSuffix(f.Name(), ".log") {
			foundBackup = true
		}
	}

	if !foundMain {
		t.Error("Arquivo principal de log não foi criado")
	}

	if !foundBackup {
		t.Error("Arquivo de backup (rotacionado) não foi criado")
	}
}

func TestLogger_MultiWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "testmulti.log")

	// Buffer para capturar a "tela"
	var buf strings.Builder

	log := NewLogger(
		WithJSON(true),
		WithMultiWriterTo(&buf, logFile, 1, 1, 1, false),
	)

	// Escreve alguns logs
	log.Info("Log multi", "user", "john", "action", "test")

	// Lê do buffer
	outStr := buf.String()
	if !strings.Contains(outStr, "\"user\":\"john\"") {
		t.Errorf("Buffer não contém log esperado: %q", outStr)
	}

	// Dá um tempinho para o lumberjack garantir flush
	time.Sleep(time.Second * 1)

	file, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("Falha ao abrir arquivo de log: %v", err)
	}
	defer file.Close()

	// Procura a mesma mensagem no arquivo
	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "\"user\":\"john\"") {
			found = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Erro ao ler arquivo de log: %v", err)
	}
	if !found {
		t.Error("Arquivo de log não contém a mensagem esperada")
	}
}
