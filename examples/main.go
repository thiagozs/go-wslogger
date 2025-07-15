package main

import (
	"context"
	"log"
	"os"

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
	logr.ErrorCtx(ctx, "Log de erro com span", "err", "Falha na conexão", "retry", 3)
	span.End()

	// Redireciona a saída para um arquivo.
	f, err := os.Create("log.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fileLogger := logger.NewLogger(
		logger.WithWriter(f),
		logger.WithFormat("[{time}] [{app_name}] [{caller}] [{level}] {message} {extra}"),
		logger.WithAppName("ExemploApp"),
	)

	fileLogger.Warn("Log escrito no arquivo", "foo", "bar")

	loggerJson := logger.NewLogger(
		logger.WithFormat("{\"time\":\"{time}\", \"app_name\":\"{app_name}\", \"caller\":\"{caller}\", \"level\":\"{level}\", \"message\":\"{message}\", \"extra\": {extra}}"),
		logger.WithAppName("ExemploApp"),
		logger.WithColor(false),
	)

	loggerJson.Warn("Log escrito no arquivo em JSON", "foo", "bar")

}
