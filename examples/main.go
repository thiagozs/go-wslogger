package main

import (
	"context"

	logger "github.com/thiagozs/go-logger"
	sdk "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Cria o logger com layout customizado.
	logr := logger.NewLogger(
		logger.WithFormat("[{time}] [{app_name}] [{caller}] [{level}] {message} {extra}"),
		logger.WithAppName("ExemploApp"),
		logger.WithColor(true),
	)

	// Exemplo sem contexto (sem OTEL).
	logr.Info("Teste de log", "err", "Error na aplicacao xyz", "foo", "bar")

	// Exemplo com contexto sem span ativo.
	ctx := context.Background()
	logr.WarnCtx(ctx, "Log de aviso", "user", "john_doe", "status", "pending approval")

	// Exemplo com span ativo usando o SDK do OTEL.
	tracer := sdk.NewTracerProvider().Tracer("example")
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

	// Examplo de logger com JSON (rotacionando e com stdout)
	loggerJsonMulti := logger.NewLogger(
		logger.WithJSON(true),
		logger.WithMultiWriter("json_log_multi.txt", 1, 3, 1, false),
		logger.WithAppName("JsonLoggerMulti"),
	)

	loggerJsonMulti.Info("Log em JSON", "user", "jane_doe", "action", "login")
	loggerJsonMulti.Warn("Log de aviso em JSON", "foo", "bar", "status", "processing")
	loggerJsonMulti.Error("Log de erro em JSON", "err", "Erro ao processar requisição", "code", 500)

}
