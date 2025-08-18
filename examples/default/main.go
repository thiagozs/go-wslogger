package main

import (
	"context"
	"fmt"

	logger "github.com/thiagozs/go-wslogger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Register a global TracerProvider for OpenTelemetry
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Println("Error shutting down TracerProvider:", err)
		}
	}()

	// Cria o logger com layout customizado.
	logr := logger.NewLogger(
		logger.WithFormat("[{time}] [{app_name}] [{caller}] [{level}] {message} {extra}"),
		logger.WithAppName("ExemploApp"),
		logger.WithColor(true),
	)

	// Exemplo sem contexto (sem OTEL).
	logr.Info("Teste de log", "err", "Error na aplicacao xyz", "foo", "bar")
	logr.Debug("Debug Teste de log", "err", "Error na aplicacao xyz", "foo", "bar")
	logr.Warn("Warning Teste de log", "warn", "User not found", "user_id", "123")
	logr.Info("--------------")
	// Exemplo com contexto sem span ativo.
	ctx := context.Background()
	logr.WarnCtx(ctx, "With CTX Log de aviso", "user", "john_doe", "status", "pending approval")

	// Exemplo com span ativo usando o tracer global do OTEL.
	tracer := otel.Tracer("example")
	ctx, span := tracer.Start(ctx, "dummySpan")
	defer span.End()
	logr.ErrorCtx(ctx, "Log de erro com SPAN", "err", "Falha na conexão", "retry", 3)

	// Exemplo de logger com JSON (somente rotacionando sem stdout).
	loggerJson := logger.NewLogger(
		logger.WithJSON(true),
		logger.WithRotatingFile("json_log.txt", 1, 3, 1, false),
		logger.WithAppName("JsonLogger"),
	)

	loggerJson.Info("Log em JSON", "user", "jane_doe", "action", "login")
	loggerJson.Warn("Log de aviso em JSON", "foo", "bar", "status", "processing")
	loggerJson.Error("Log de erro em JSON", "err", "Erro ao processar requisição", "code", 500)
	loggerJson.Debug("Debug Log em JSON", "user", "jane_doe", "action", "login")

	// Examplo de logger com JSON (rotacionando e com stdout)
	loggerJsonMulti := logger.NewLogger(
		logger.WithJSON(true),
		logger.WithMultiWriter("json_log_multi.txt", 1, 3, 1, false),
		logger.WithAppName("JsonLoggerMulti"),
	)

	loggerJsonMulti.Info("Log em JSON", "user", "jane_doe", "action", "login")
	loggerJsonMulti.Warn("Log de aviso em JSON", "foo", "bar", "status", "processing")
	loggerJsonMulti.Error("Log de erro em JSON", "err", "Erro ao processar requisição", "code", 500)
	loggerJsonMulti.Debug("Log em JSON", "user", "jane_doe", "action", "login")

	// Exemplo de uso do wslogger com OpenTelemetry e contexto.
	log := logger.NewLogger(
		logger.WithAppName("otel-demo"),
		logger.WithColor(true),
		logger.WithFormat("[{time}] [{app_name}] [{caller}] [{level}] {message} {extra}"),
	)

	tracerOt := otel.Tracer("example-tracer")
	ctx, spanOt := tracerOt.Start(context.Background(), "main-span")
	defer spanOt.End()

	log.InfoCtx(ctx, "Log com contexto OTel")
	log.WarnCtx(ctx, "Log com contexto OTel", "oi", "teste")
	log.ErrorCtxf(ctx, "Erro com contexto e wrap: %w", fmt.Errorf("erro original"))
	log.DebugCtx(ctx, "Log de depuração com contexto", "foo", "bar")

	// Log normal
	log.Infof("Mensagem formatada: %d + %d = %d", 2, 3, 5)

	log.Infof("Mensagem formatada: %d + %d = %d", 2, 3, 5)
	log.Debugf("Mensagem formatada: %d + %d = %d", 2, 3, 5)

}
