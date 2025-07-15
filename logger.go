package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Option define uma função de configuração para o Logger.
type Option func(*Logger)

// Logger é o nosso logger customizado.
type Logger struct {
	writer  io.Writer
	format  string // Layout de formatação com placeholders.
	appName string
	color   bool
}

// Valores default.
const (
	defaultFormat  = "[{time}] [{app_name}] [{caller}] [{level}] {message}{extra}"
	defaultAppName = "MyApp"
)

// Códigos ANSI para cores.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// NewLogger cria um novo Logger com as opções fornecidas.
func NewLogger(opts ...Option) *Logger {
	l := &Logger{
		writer:  os.Stdout,
		format:  defaultFormat,
		appName: defaultAppName,
		color:   true,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithWriter define um io.Writer customizado (por exemplo, para testes).
func WithWriter(w io.Writer) Option {
	return func(l *Logger) {
		l.writer = w
	}
}

// WithFormat define um layout customizado para os logs.
func WithFormat(format string) Option {
	return func(l *Logger) {
		if format != "" {
			l.format = format
		}
	}
}

// WithAppName define o nome da aplicação.
func WithAppName(name string) Option {
	return func(l *Logger) {
		l.appName = name
	}
}

// WithColor habilita ou desabilita o uso de cores.
func WithColor(enable bool) Option {
	return func(l *Logger) {
		l.color = enable
	}
}

// KeyValuePair representa um par chave-valor.
type KeyValuePair struct {
	key   string
	value string
}

// parseLogArgs extrai a mensagem principal e os pares chave-valor (se houver).
// O primeiro parâmetro é a mensagem e os demais devem vir em pares.
func parseLogArgs(args ...interface{}) (string, []KeyValuePair) {
	if len(args) == 0 {
		return "", nil
	}
	mainMsg := fmt.Sprint(args[0])
	var extras []KeyValuePair
	n := len(args)
	for i := 1; i+1 < n; i += 2 {
		key := fmt.Sprint(args[i])
		value := formatValue(args[i+1])
		extras = append(extras, KeyValuePair{key, value})
	}
	return mainMsg, extras
}

// formatValue formata um valor: se for string
// e contiver espaço, envolve em aspas.
func formatValue(v any) string {
	s, ok := v.(string)
	if ok {
		if strings.Contains(s, " ") {
			return fmt.Sprintf("\"%s\"", s)
		}
		return s
	}
	return fmt.Sprint(v)
}

// getColorCode retorna o código ANSI de cor para um nível (sem reset).
func getColorCode(level string) string {
	switch level {
	case "INFO":
		return colorGreen
	case "WARN":
		return colorYellow
	case "ERROR":
		return colorRed
	case "DEBUG":
		return colorCyan
	default:
		return ""
	}
}

// formatMessage constrói a mensagem final substituindo os placeholders.
// Placeholders suportados: {time}, {app_name}, {caller}, {level},
// {message}, {trace_id}, {span_id} e {extra}.
func (l *Logger) formatMessage(level, msg, extra string, t time.Time,
	traceID, spanID, caller string) string {
	// Aplica cor ao nível, se habilitado.
	colorCode := ""
	if l.color {
		colorCode = getColorCode(level)
		level = colorCode + level + colorReset
	}

	// Se os IDs estiverem preenchidos, monta as strings com os prefixos.
	if traceID != "" {
		traceID = "trace_id=" + traceID
	}
	if spanID != "" {
		spanID = "span_id=" + spanID
	}

	replacements := map[string]string{
		"{time}":     t.Format("2006-01-02 15:04:05"),
		"{app_name}": l.appName,
		"{caller}":   caller,
		"{level}":    level,
		"{message}":  msg,
		"{trace_id}": traceID,
		"{span_id}":  spanID,
		"{extra}":    extra,
	}

	formatted := l.format
	for placeholder, value := range replacements {
		if value == "" {
			continue // ignora se o valor estiver vazio
		}

		formatted = strings.ReplaceAll(formatted, placeholder, value)
	}
	return formatted
}

// getCaller retorna uma string com o arquivo, linha e função que chamou o log.
func (l *Logger) getCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	fileBase := path.Base(file)
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = path.Base(fn.Name())
	}
	return fmt.Sprintf("%s:%d,%s", fileBase, line, funcName)
}

// logInternal realiza o log: extrai trace, caller,
// processa os extras e envia a mensagem formatada.
func (l *Logger) logInternal(level, msg string,
	extras []KeyValuePair, ctx context.Context) {
	now := time.Now()

	// Extrai dados de tracing, se houver.
	var traceID, spanID string
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceID = sc.TraceID().String()
			spanID = sc.SpanID().String()
		}
	}

	// Extrai informações do caller;
	// skip ajustado para obter o chamador real.
	caller := l.getCaller(6)

	// Formata os extras, colorindo as chaves com o mesmo código do nível.
	extraStr := ""
	if len(extras) > 0 {
		var parts []string
		colorCode := ""
		if l.color {
			colorCode = getColorCode(level)
		}
		for _, kv := range extras {
			// A chave é colorida.
			keyColored := kv.key
			if colorCode != "" {
				keyColored = colorCode + kv.key + colorReset
			}
			parts = append(parts, fmt.Sprintf("%s=%s", keyColored, kv.value))
		}

		// Adiciona um único espaço antes dos extras.
		extraStr = strings.Join(parts, " ")
	}

	output := l.formatMessage(level, msg, extraStr, now, traceID, spanID, caller)
	fmt.Fprintln(l.writer, output)
}

// Métodos de log sem contexto.
func (l *Logger) Info(args ...any)  { l.logWithArgs("INFO", args, context.Background()) }
func (l *Logger) Warn(args ...any)  { l.logWithArgs("WARN", args, context.Background()) }
func (l *Logger) Error(args ...any) { l.logWithArgs("ERROR", args, context.Background()) }
func (l *Logger) Debug(args ...any) { l.logWithArgs("DEBUG", args, context.Background()) }

// Métodos de log com contexto.
func (l *Logger) InfoCtx(ctx context.Context, args ...any) { l.logWithArgs("INFO", args, ctx) }
func (l *Logger) WarnCtx(ctx context.Context, args ...any) { l.logWithArgs("WARN", args, ctx) }
func (l *Logger) ErrorCtx(ctx context.Context, args ...any) {
	l.logWithArgs("ERROR", args, ctx)
}
func (l *Logger) DebugCtx(ctx context.Context, args ...any) {
	l.logWithArgs("DEBUG", args, ctx)
}

// logWithArgs processa os argumentos e chama o log interno.
func (l *Logger) logWithArgs(level string, args []any, ctx context.Context) {
	msg, extras := parseLogArgs(args...)
	l.logInternal(level, msg, extras, ctx)
}
