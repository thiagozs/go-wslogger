package wslogger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Option define uma função de configuração para o Logger.
type Option func(*Logger)

// Logger customizado.
type Logger struct {
	writer           io.Writer
	format           string
	appName          string
	color            bool
	jsonMode         bool
	includeSpanAttrs bool
}

// WithWriter permite configurar o destino de saída do logger.
func WithWriter(w io.Writer) Option {
	return func(l *Logger) {
		if w != nil {
			l.writer = w
		}
	}
}

// Flags para formato do caller
const (
	CallerFlagFull   uint8 = iota // função,arquivo:linha
	CallerFlagFunc                // função
	CallerFlagFcLine              // função:linha
	CallerFlagPkg                 // pacote
	CallerFlagPkgFnl              // pacote,arquivo:linha
	CallerFlagFnlFcn              // arquivo:linha,função
	CallerFlagFnLine              // arquivo:linha
	CallerFlagFcName              // nome da função
	CallerFlagFpLine              // caminho/arquivo:linha
)

type CallerFormatFn func(*runtime.Frame) string

// Valores default.
const (
	defaultFormat  = "[{time}] [{app_name}] [{caller}] [{level}] {message} {extra}"
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

// Funções utilitárias para caller e JSON
func getCallerFrame(skip int) (runtime.Frame, bool) {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return runtime.Frame{}, false
	}
	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()
	return frame, true
}

func formatCaller(rf *runtime.Frame, flag uint8, custom CallerFormatFn) string {
	lineNum := fmt.Sprintf("%d", rf.Line)
	if custom != nil {
		return custom(rf)
	}
	switch flag {
	case CallerFlagFull:
		return rf.Function + "," + rf.File + ":" + lineNum
	case CallerFlagFunc:
		return rf.Function
	case CallerFlagFcLine:
		return rf.Function + ":" + lineNum
	case CallerFlagPkg:
		i := strings.LastIndex(rf.Function, "/")
		if i == -1 {
			return rf.Function
		}
		return rf.Function[:i]
	case CallerFlagPkgFnl:
		// tenta extrair pacote + arquivo:linha
		i := strings.LastIndex(rf.Function, "/")
		if i == -1 {
			i = 0
		}
		j := strings.IndexByte(rf.Function[i+1:], '.')
		if j >= 0 {
			i += j + 1
		}
		return rf.Function[:i+1] + "," + filepath.Base(rf.File) + ":" + lineNum
	case CallerFlagFnlFcn:
		ss := strings.Split(rf.Function, ".")
		fn := ss[len(ss)-1]
		return filepath.Base(rf.File) + ":" + lineNum + "," + fn
	case CallerFlagFnLine:
		return filepath.Base(rf.File) + ":" + lineNum
	case CallerFlagFcName:
		ss := strings.Split(rf.Function, ".")
		return ss[len(ss)-1]
	default: // CallerFlagFpLine
		return rf.File + ":" + lineNum
	}
}

func (l *Logger) logInternalJSON(level, msg string, extras []KeyValuePair, ctx context.Context) {
	now := time.Now()
	var traceID, spanID string
	var extraMap map[string]string
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceID = sc.TraceID().String()
			spanID = sc.SpanID().String()
		}
		if l.includeSpanAttrs {
			extraMap = spanAttributesToMap(span)
		}
	}
	if extraMap == nil {
		extraMap = make(map[string]string)
	}
	for _, kv := range extras {
		extraMap[kv.key] = kv.value
	}
	// Se o criador da goroutine passou o local como extra, usa ele
	caller := ""
	for _, kv := range extras {
		if kv.key == "goroutine_caller" {
			caller = kv.value
			break
		}
	}
	if caller == "" {
		caller = l.getCaller(6)
	}
	record := logJSON{
		Time:    now.Format("2006-01-02 15:04:05"),
		Level:   level,
		App:     l.appName,
		Caller:  caller,
		Message: msg,
		TraceID: traceID,
		SpanID:  spanID,
		Extra:   extraMap,
	}
	data, _ := json.Marshal(record)
	fmt.Fprintln(l.writer, string(data))
}

// ==== Options ======

func NewLogger(opts ...Option) *Logger {
	l := &Logger{
		writer:  os.Stdout,
		format:  defaultFormat,
		appName: defaultAppName,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func WithAppName(name string) Option {
	return func(l *Logger) { l.appName = name }
}

func WithColor(enable bool) Option {
	return func(l *Logger) { l.color = enable }
}

func WithJSON(enable bool) Option {
	return func(l *Logger) { l.jsonMode = enable }
}

// WithFormat permite configurar o template de saída do logger.
func WithFormat(format string) Option {
	return func(l *Logger) { l.format = format }
}

// Ativa/desativa captura automática de atributos do span OTel
func WithSpanAttributes(enable bool) Option {
	return func(l *Logger) { l.includeSpanAttrs = enable }
}

func WithRotatingFile(filename string, maxSizeMB, maxBackups, maxAgeDays int, compress bool) Option {
	return func(l *Logger) {
		l.writer = &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSizeMB,  // megabytes por arquivo
			MaxBackups: maxBackups, // quantos arquivos manter
			MaxAge:     maxAgeDays, // dias até rodar
			Compress:   compress,   // gzip rotacionados
		}
	}
}

func WithMultiWriter(filename string, maxSizeMB, maxBackups, maxAgeDays int, compress bool) Option {
	return WithMultiWriterTo(os.Stdout, filename, maxSizeMB, maxBackups, maxAgeDays, compress)
}

func WithMultiWriterTo(w io.Writer, filename string, maxSizeMB, maxBackups, maxAgeDays int, compress bool) Option {
	return func(l *Logger) {
		l.writer = io.MultiWriter(
			w,
			&lumberjack.Logger{
				Filename:   filename,
				MaxSize:    maxSizeMB,
				MaxBackups: maxBackups,
				MaxAge:     maxAgeDays,
				Compress:   compress,
			},
		)
	}
}

// KeyValuePair representa um par chave-valor.
type KeyValuePair struct {
	key   string
	value string
}

func parseLogArgs(args ...any) (string, []KeyValuePair) {
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

func (l *Logger) formatMessage(level, msg, extra string, t time.Time,
	traceID, spanID, caller string) string {

	colorCode := ""
	if l.color {
		colorCode = getColorCode(level)
		level = colorCode + level + colorReset
	}

	// Aplica cor apenas ao valor de trace_id/span_id se colorido
	if traceID != "" {
		val := strings.TrimPrefix(traceID, "trace_id=")
		if l.color {
			traceID = colorCode + "trace_id" + colorCode + colorReset + "=" + val
		} else {
			traceID = "trace_id=" + val
		}
	}
	if spanID != "" {
		val := strings.TrimPrefix(spanID, "span_id=")
		if l.color {
			spanID = colorCode + "span_id" + colorCode + colorReset + "=" + val
		} else {
			spanID = "span_id=" + val
		}
	}

	replacements := map[string]string{
		"{time}":     t.Format("2006-01-02 15:04:05"),
		"{app_name}": l.appName,
		"{caller}":   caller,
		"{level}":    level,
		"{message}":  msg,
		"{trace_id}": traceID,
		"{span_id}":  spanID,
	}

	formatted := l.format
	for placeholder, value := range replacements {
		if value == "" {
			continue
		}
		formatted = strings.ReplaceAll(formatted, placeholder, value)
	}

	// Remove {extra} se não houver extras
	if extra != "" {
		formatted = strings.ReplaceAll(formatted, "{extra}", extra)
	} else {
		formatted = strings.ReplaceAll(formatted, "{extra}", "")
		formatted = formatted[:len(formatted)-1]
	}

	// Garante que trace_id/span_id sempre aparecem se OTel estiver presente
	// (ou seja, se traceID ou spanID não estão vazios)
	appendFields := []string{}
	if traceID != "" || spanID != "" {
		hasTrace := strings.Contains(formatted, "trace_id=")
		hasSpan := strings.Contains(formatted, "span_id=")
		if !hasTrace && traceID != "" {
			appendFields = append(appendFields, traceID)
		}
		if !hasSpan && spanID != "" {
			appendFields = append(appendFields, spanID)
		}
		if len(appendFields) > 0 {
			// Remove todos os espaços extras do final
			formatted = strings.TrimRight(formatted, " ")
			// Se não terminar com espaço, adiciona um
			if len(formatted) > 0 && formatted[len(formatted)-1] != ' ' {
				formatted += " "
			}
			formatted += strings.Join(appendFields, " ")
		}
	}

	return formatted
}

// getCaller com suporte a flags e função customizada
func (l *Logger) getCaller(skip int) string {
	frame, ok := getCallerFrame(skip)
	if !ok {
		return "unknown"
	}
	// Você pode expor o flag como opção do Logger, aqui uso FnLine por padrão
	return formatCaller(&frame, CallerFlagFnLine, nil)
}

// JSON struct para output
type logJSON struct {
	Time    string            `json:"time"`
	Level   string            `json:"level"`
	App     string            `json:"app_name"`
	Caller  string            `json:"caller"`
	Message string            `json:"message"`
	TraceID string            `json:"trace_id,omitempty"`
	SpanID  string            `json:"span_id,omitempty"`
	Extra   map[string]string `json:"extra,omitempty"`
}

// Captura atributos do Span OTel para map[string]string
func spanAttributesToMap(span trace.Span) map[string]string {
	out := make(map[string]string)
	if s, ok := span.(interface{ Attributes() []attribute.KeyValue }); ok {
		for _, attr := range s.Attributes() {
			out[string(attr.Key)] = attr.Value.Emit()
		}
	}
	return out
}

// ====== logInternal SWITCH ======
func (l *Logger) logInternal(level, msg string,
	extras []KeyValuePair, ctx context.Context) {
	if l.jsonMode {
		l.logInternalJSON(level, msg, extras, ctx)
		return
	}
	now := time.Now()

	var traceID, spanID string
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceID = sc.TraceID().String()
			spanID = sc.SpanID().String()
		}
	}
	// permite que criadores de goroutine passem o caller como extra
	caller := ""
	for _, kv := range extras {
		if kv.key == "goroutine_caller" {
			caller = kv.value
			break
		}
	}
	if caller == "" {
		caller = l.getCaller(6)
	}

	extraStr := ""
	if len(extras) > 0 {
		var parts []string
		colorCode := ""
		if l.color {
			colorCode = getColorCode(level)
		}
		for _, kv := range extras {
			keyColored := kv.key
			if colorCode != "" {
				keyColored = colorCode + kv.key + colorReset
			}
			parts = append(parts, fmt.Sprintf("%s=%s", keyColored, kv.value))
		}
		extraStr = strings.Join(parts, " ")
	}
	output := l.formatMessage(level, msg, extraStr, now, traceID, spanID, caller)
	fmt.Fprintln(l.writer, output)
}

func (l *Logger) SetAppName(name string) {
	l.appName = name
}

func (l *Logger) SetColor(enabled bool) {
	l.color = enabled
}

func (l *Logger) SetJSON(enabled bool) {
	l.jsonMode = enabled
}

func (l *Logger) SetIncludeSpanAttrs(enabled bool) {
	l.includeSpanAttrs = enabled
}

// Métodos de log sem contexto.

// Métodos de log com formatação estilo fmt.Sprintf
func (l *Logger) Infof(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("INFO", []any{msg}, context.Background())
}
func (l *Logger) Warnf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("WARN", []any{msg}, context.Background())
}
func (l *Logger) Errorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("ERROR", []any{msg}, context.Background())
}
func (l *Logger) Debugf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("DEBUG", []any{msg}, context.Background())
}

func (l *Logger) Info(args ...any)  { l.logWithArgs("INFO", args, context.Background()) }
func (l *Logger) Warn(args ...any)  { l.logWithArgs("WARN", args, context.Background()) }
func (l *Logger) Error(args ...any) { l.logWithArgs("ERROR", args, context.Background()) }
func (l *Logger) Debug(args ...any) { l.logWithArgs("DEBUG", args, context.Background()) }

// Métodos de log com contexto.
func (l *Logger) InfoCtx(ctx context.Context, args ...any)  { l.logWithArgs("INFO", args, ctx) }
func (l *Logger) WarnCtx(ctx context.Context, args ...any)  { l.logWithArgs("WARN", args, ctx) }
func (l *Logger) ErrorCtx(ctx context.Context, args ...any) { l.logWithArgs("ERROR", args, ctx) }
func (l *Logger) DebugCtx(ctx context.Context, args ...any) { l.logWithArgs("DEBUG", args, ctx) }

func (l *Logger) InfoCtxf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("INFO", []any{msg}, ctx)
}
func (l *Logger) WarnCtxf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("WARN", []any{msg}, ctx)
}
func (l *Logger) ErrorCtxf(ctx context.Context, format string, args ...any) {
	if strings.Contains(format, "%w") {
		msg := fmt.Errorf(format, args...).Error()
		l.logWithArgs("ERROR", []any{msg}, ctx)
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.logWithArgs("ERROR", []any{msg}, ctx)
}
func (l *Logger) DebugCtxf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if strings.Contains(format, "%w") {
		msg = fmt.Errorf(format, args...).Error()
	}
	l.logWithArgs("DEBUG", []any{msg}, ctx)
}

func (l *Logger) logWithArgs(level string, args []any, ctx context.Context) {
	msg, extras := parseLogArgs(args...)
	l.logInternal(level, msg, extras, ctx)
}
