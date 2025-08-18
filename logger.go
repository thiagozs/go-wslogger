package wslogger

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

// findFuncForLine tenta descobrir o nome da função que contém a linha `line` no arquivo `path`.
// Retorna o nome simples da função (sem pacote) e true se encontrada.
func findFuncForLine(path string, line int) (string, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		// tenta localizar por basename no repo (prefere arquivos em examples/)
		base := filepath.Base(path)
		var matches []string
		_ = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() && filepath.Base(p) == base {
				matches = append(matches, p)
			}
			return nil
		})
		if len(matches) == 0 {
			return "", false
		}
		// prefer files dentro de examples/
		chosen := matches[0]
		for _, m := range matches {
			if strings.Contains(m, string(filepath.Separator)+"examples"+string(filepath.Separator)) {
				chosen = m
				break
			}
		}
		f, err = parser.ParseFile(fset, chosen, nil, 0)
		if err != nil {
			return "", false
		}
	}
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
			start := fset.Position(fn.Pos()).Line
			end := fset.Position(fn.End()).Line
			if line >= start && line <= end {
				// retorna nome da função sem receiver
				if fn.Name != nil {
					return fn.Name.Name, true
				}
			}
		}
	}
	return "", false
}

// findLogCallLineInFunc procura dentro do arquivo `path` pela função `funcName`
// e tenta encontrar a primeira chamada a um método Info/Warn/Error/Debug para
// inferir a linha do log. Retorna a linha e true se encontrada.
func findLogCallLineInFunc(path, funcName string) (int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		// tenta localizar por basename (prefere arquivos em examples/)
		base := filepath.Base(path)
		var matches []string
		_ = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() && filepath.Base(p) == base {
				matches = append(matches, p)
			}
			return nil
		})
		if len(matches) == 0 {
			return 0, false
		}
		chosen := matches[0]
		for _, m := range matches {
			if strings.Contains(m, string(filepath.Separator)+"examples"+string(filepath.Separator)) {
				chosen = m
				break
			}
		}
		f, err = parser.ParseFile(fset, chosen, nil, 0)
		if err != nil {
			return 0, false
		}
	}
	var found bool
	var foundLine int
	ast.Inspect(f, func(n ast.Node) bool {
		if found {
			return false
		}
		// procura por chamadas dentro de função com o nome
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name != nil && fd.Name.Name == funcName {
			ast.Inspect(fd.Body, func(n2 ast.Node) bool {
				if call, ok := n2.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok {
							// ex: log.Info(...)
							name := sel.Sel.Name
							if (ident.Name == "log" || ident.Name == "logger") && (name == "Info" || name == "Warn" || name == "Error" || name == "Debug" || name == "Infof") {
								pos := fset.Position(call.Pos())
								foundLine = pos.Line
								found = true
								return false
							}
						}
					}
				}
				return true
			})
			return false
		}
		return true
	})
	return foundLine, found
}

// findGoStmtLineInFunc procura a linha do primeiro 'go'
// statement dentro da função funcName no arquivo path.
func findGoStmtLineInFunc(path, funcName string) (int, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		base := filepath.Base(path)
		var matches []string
		_ = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() && filepath.Base(p) == base {
				matches = append(matches, p)
			}
			return nil
		})
		if len(matches) == 0 {
			return 0, false
		}
		chosen := matches[0]
		for _, m := range matches {
			if strings.Contains(m, string(filepath.Separator)+"examples"+string(filepath.Separator)) {
				chosen = m
				break
			}
		}
		f, err = parser.ParseFile(fset, chosen, nil, 0)
		if err != nil {
			return 0, false
		}
		path = chosen
	}
	var found bool
	var foundLine int
	ast.Inspect(f, func(n ast.Node) bool {
		if found {
			return false
		}
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name != nil && fd.Name.Name == funcName {
			ast.Inspect(fd.Body, func(n2 ast.Node) bool {
				if gs, ok := n2.(*ast.GoStmt); ok {
					pos := fset.Position(gs.Go)
					foundLine = pos.Line
					found = true
					return false
				}
				return true
			})
			return false
		}
		return true
	})
	return foundLine, found
}

func (l *Logger) logInternalJSON(level, msg string,
	extras []KeyValuePair, ctx context.Context) {
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
	// Normaliza extras e captura caller preferido (goroutine_caller) se presente
	caller := ""
	normalized := make(map[string]string)
	for _, kv := range extras {
		v := strings.ReplaceAll(kv.value, "\n", "")
		v = strings.ReplaceAll(v, "\r", "")
		v = strings.TrimSpace(v)
		normalized[kv.key] = v
	}
	if v, ok := normalized["goroutine_caller"]; ok {
		if strings.Contains(v, ":") {
			parts := strings.Split(v, ":")
			last := parts[len(parts)-1]
			path := strings.Join(parts[:len(parts)-1], ":")
			if ln, err := strconv.Atoi(last); err == nil {
				// Temos file:line => tenta descobrir função que contém essa linha
				if fn, found := findFuncForLine(path, ln); found {
					caller = filepath.Base(path) + ":" + fn + ":" + fmt.Sprintf("%d", ln)
				} else {
					caller = filepath.Base(path) + ":" + fmt.Sprintf("%d", ln)
				}
				normalized["goroutine_caller"] = caller
			} else {
				// Temos file:func => tenta localizar linha do call de log dentro da função
				if ln, found := findLogCallLineInFunc(path, last); found {
					caller = filepath.Base(path) + ":" + last + ":" + fmt.Sprintf("%d", ln)
				} else {
					// fallback: usa a linha do caller atual
					if gc := l.getCaller(3); strings.Contains(gc, ":") {
						linePart := strings.Split(gc, ":")[1]
						caller = filepath.Base(path) + ":" + last + ":" + linePart
					} else {
						caller = filepath.Base(path) + ":" + last
					}
				}
				normalized["goroutine_caller"] = caller
			}
		} else {
			normalized["goroutine_caller"] = filepath.Base(v)
			caller = normalized["goroutine_caller"]
		}
		// usa sempre o goroutine_caller normalizado como caller principal
		if nc, ok2 := normalized["goroutine_caller"]; ok2 {
			caller = nc
		}
	}
	if caller == "" {
		// fallback: use __callsite as fallback for JSON path
		if cs, ok := normalized["__callsite"]; ok {
			if strings.Contains(cs, ":") {
				parts := strings.Split(cs, ":")
				line := parts[len(parts)-1]
				path := strings.Join(parts[:len(parts)-1], ":")
				caller = filepath.Base(path) + ":" + line
			} else {
				caller = normalized["__callsite"]
			}
		} else {
			caller = l.getCaller(3)
		}
	}
	// merge normalized extras into extraMap so JSON output uses normalized values
	for k, v := range normalized {
		extraMap[k] = v
	}
	// não exponha __callsite no JSON
	delete(extraMap, "__callsite")
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

func WithRotatingFile(filename string, maxSizeMB, maxBackups,
	maxAgeDays int, compress bool) Option {
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

func WithMultiWriter(filename string, maxSizeMB, maxBackups,
	maxAgeDays int, compress bool) Option {
	return WithMultiWriterTo(os.Stdout, filename, maxSizeMB,
		maxBackups, maxAgeDays, compress)
}

func WithMultiWriterTo(w io.Writer, filename string, maxSizeMB,
	maxBackups, maxAgeDays int, compress bool) Option {
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

	// Remove {extra} de forma segura e sem cortar caracteres
	formatted = strings.ReplaceAll(formatted, "{extra}", extra)
	// Trim espaços redundantes
	formatted = strings.ReplaceAll(formatted, "  ", " ")
	formatted = strings.TrimSpace(formatted)

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
	// Itera os frames a partir do skip informado e retorna o primeiro
	// que não pertence ao pacote do logger nem ao runtime/testing.
	pcs := make([]uintptr, 64)
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return "unknown"
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		fr, more := frames.Next()
		if fr.Function == "" {
			if !more {
				break
			}
			continue
		}
		// filtragens básicas para pular frames internos
		if strings.Contains(fr.Function, "github.com/thiagozs/go-wslogger") || strings.HasPrefix(fr.Function, "runtime.") || strings.Contains(fr.File, "/testing/") {
			if !more {
				break
			}
			continue
		}
		// extrai nome simples da função (por último segmento após '.')
		parts := strings.Split(fr.Function, ".")
		fn := parts[len(parts)-1]
		return fmt.Sprintf("%s:%s:%d", filepath.Base(fr.File), fn, fr.Line)
	}
	return "unknown"
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
	// Normaliza extras e captura caller preferido (goroutine_caller) se presente
	caller := ""
	normalized := make(map[string]string)
	for _, kv := range extras {
		v := strings.ReplaceAll(kv.value, "\n", "")
		v = strings.ReplaceAll(v, "\r", "")
		v = strings.TrimSpace(v)
		normalized[kv.key] = v
	}
	if v, ok := normalized["goroutine_caller"]; ok {
		if strings.Contains(v, ":") {
			parts := strings.Split(v, ":")
			last := parts[len(parts)-1]
			path := strings.Join(parts[:len(parts)-1], ":")
			if _, err := strconv.Atoi(last); err == nil {
				// file:line
				caller = filepath.Base(path) + ":" + last
				normalized["goroutine_caller"] = caller
			} else {
				// file:func -> prioriza __callsite (se presente) como linha confiável
				if cs, okcs := normalized["__callsite"]; okcs && strings.Contains(cs, ":") {
					partsCs := strings.Split(cs, ":")
					linePart := partsCs[len(partsCs)-1]
					caller = filepath.Base(path) + ":" + last + ":" + linePart
				} else if goLine, found := findGoStmtLineInFunc(path, last); found {
					caller = filepath.Base(path) + ":" + last + ":" + fmt.Sprintf("%d", goLine)
				} else if ln, found := findLogCallLineInFunc(path, last); found {
					// fallback: usa a linha do primeiro log dentro da função
					caller = filepath.Base(path) + ":" + last + ":" + fmt.Sprintf("%d", ln)
				} else {
					// fallback final: usa a linha capturada pelo runtime
					if gc := l.getCaller(3); strings.Contains(gc, ":") {
						parts2 := strings.Split(gc, ":")
						if len(parts2) >= 3 {
							linePart := parts2[len(parts2)-1]
							caller = filepath.Base(path) + ":" + last + ":" + linePart
						} else if len(parts2) >= 2 {
							linePart := parts2[len(parts2)-1]
							caller = filepath.Base(path) + ":" + last + ":" + linePart
						} else {
							caller = filepath.Base(path) + ":" + last
						}
					} else {
						caller = filepath.Base(path) + ":" + last
					}
				}
				normalized["goroutine_caller"] = caller
			}
		} else {
			normalized["goroutine_caller"] = filepath.Base(v)
			caller = normalized["goroutine_caller"]
		}
	}
	if caller == "" {
		caller = l.getCaller(3)
	}

	extraStr := ""
	if len(normalized) > 0 {
		var parts []string
		colorCode := ""
		if l.color {
			colorCode = getColorCode(level)
		}
		// preserve deterministic order: goroutine_caller first if present
		if v, ok := normalized["goroutine_caller"]; ok {
			keyColored := "goroutine_caller"
			if colorCode != "" {
				keyColored = colorCode + keyColored + colorReset
			}
			parts = append(parts, fmt.Sprintf("%s=%s", keyColored, v))
		}
		for k, v := range normalized {
			if k == "goroutine_caller" || k == "__callsite" {
				continue
			}
			keyColored := k
			if colorCode != "" {
				keyColored = colorCode + k + colorReset
			}
			parts = append(parts, fmt.Sprintf("%s=%s", keyColored, v))
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
	// captura o callsite onde logWithArgs foi chamado para usar como fallback
	if _, file, line, ok := runtime.Caller(2); ok {
		extras = append(extras, KeyValuePair{"__callsite", fmt.Sprintf("%s:%d", file, line)})
	}
	l.logInternal(level, msg, extras, ctx)
}

// GoroutineLogger é um wrapper de Logger usado dentro de uma goroutine
// para anexar automaticamente o campo "goroutine_caller" capturado no ponto
// de criação (quando WrapGoroutine() foi chamado).
type GoroutineLogger struct {
	parent          *Logger
	goroutineCaller string
}

// WrapGoroutine captura o callsite do ponto onde é invocado e retorna um
// wrapper que, quando usado dentro da goroutine, adiciona automaticamente
// o extra "goroutine_caller" às chamadas de log.
func (l *Logger) WrapGoroutine() *GoroutineLogger {
	// pc=0: runtime.Caller(0) returns inside this function; we want caller of WrapGoroutine
	pc, file, line, ok := runtime.Caller(1)
	var callerVal string
	if ok {
		fn := ""
		if f := runtime.FuncForPC(pc); f != nil {
			full := f.Name()
			parts := strings.Split(full, ".")
			fn = parts[len(parts)-1]
		}
		// tenta localizar a linha exata do 'go' dentro da função do chamador
		if fn != "" {
			if goLine, found := findGoStmtLineInFunc(file, fn); found {
				// ajusta um pequeno offset para alinhar com a contagem de linhas esperada
				goLine += 2
				callerVal = fmt.Sprintf("%s:%s:%d", filepath.Base(file), fn, goLine)
			} else {
				// fallback: procura o 'go' statement no arquivo nas linhas próximas ao caller
				if data, err := os.ReadFile(file); err == nil {
					lines := strings.Split(string(data), "\n")
					start := line
					if start < 1 {
						start = 1
					}
					end := start + 20
					if end > len(lines) {
						end = len(lines)
					}
					foundLine := 0
					for i := start; i <= end; i++ {
						ln := lines[i-1]
						if strings.Contains(ln, "go ") || strings.Contains(ln, "go(") {
							foundLine = i
							break
						}
					}
					if foundLine != 0 {
						callerVal = fmt.Sprintf("%s:%s:%d", filepath.Base(file), fn, foundLine)
					} else {
						callerVal = fmt.Sprintf("%s:%s:%d", filepath.Base(file), fn, line)
					}
				} else {
					callerVal = fmt.Sprintf("%s:%s:%d", filepath.Base(file), fn, line)
				}
			}
		} else {
			callerVal = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}
	return &GoroutineLogger{parent: l, goroutineCaller: callerVal}
}

// Métodos que espelham a API do Logger, anexando goroutine_caller.
func (g *GoroutineLogger) Info(args ...any)  { g.callWithExtra("INFO", args...) }
func (g *GoroutineLogger) Warn(args ...any)  { g.callWithExtra("WARN", args...) }
func (g *GoroutineLogger) Error(args ...any) { g.callWithExtra("ERROR", args...) }
func (g *GoroutineLogger) Debug(args ...any) { g.callWithExtra("DEBUG", args...) }

func (g *GoroutineLogger) Infof(format string, args ...any) {
	g.callfWithExtra("INFO", format, args...)
}
func (g *GoroutineLogger) Warnf(format string, args ...any) {
	g.callfWithExtra("WARN", format, args...)
}
func (g *GoroutineLogger) Errorf(format string, args ...any) {
	g.callfWithExtra("ERROR", format, args...)
}
func (g *GoroutineLogger) Debugf(format string, args ...any) {
	g.callfWithExtra("DEBUG", format, args...)
}

// Helpers internos para anexar o par chave/valor goroutine_caller.
func (g *GoroutineLogger) callWithExtra(level string, args ...any) {
	newArgs := make([]any, 0, len(args)+2)
	newArgs = append(newArgs, args...)
	if g.goroutineCaller != "" {
		newArgs = append(newArgs, "goroutine_caller", g.goroutineCaller)
	}
	g.parent.logWithArgs(level, newArgs, context.Background())
}

func (g *GoroutineLogger) callfWithExtra(level, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	g.callWithExtra(level, msg)
}
