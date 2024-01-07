package wslogger

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
)

type WsLogger struct {
	params *WsLoggerParams
	log    *slog.Logger
	buf    *bytes.Buffer
}

type Value = slog.Value

func NewWsLogger(opts ...Options) (*WsLogger, error) {
	params, err := newWsLoggerParams(opts...)

	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	var handler slog.Handler

	if params.GetBuffered() {
		handler = slog.NewTextHandler(buf, nil)
	} else {
		switch params.GetKind() {
		case Default:
			handler = slog.NewTextHandler(os.Stdout, nil)
		case Stdout:
			handler = slog.NewTextHandler(os.Stdout, nil)
		case File:
			//TODO: log file rotation
			file, err := os.OpenFile(params.GetFileName(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}
			handler = slog.NewTextHandler(file, nil)
		case JSON:
			handler = slog.NewJSONHandler(os.Stdout, nil)
		default:
			return nil, errors.New("invalid logger kind")
		}
	}

	logger := slog.New(handler)

	return &WsLogger{
		params: params,
		log:    logger,
		buf:    buf,
	}, nil
}

func (w *WsLogger) Info(msg string) {
	w.log.Info(msg)
}

func (w *WsLogger) Debug(msg string) {
	//REFS: the WithGroup method in slog.Handler not interact with the log
	// directly, so we need to write the group name to the buffer manually
	if w.params.GetBuffered() {
		w.buf.WriteString(msg)
	}
	w.log.Debug(msg)
}

func (w *WsLogger) Error(msg string) {
	w.log.Error(msg)
}

func (w *WsLogger) Warn(msg string) {
	w.log.Warn(msg)
}

func (w *WsLogger) WithAttrs(attrs map[string]interface{}) *WsLogger {
	var slogAttrs []slog.Attr
	for k, v := range attrs {
		slogAttr := slog.Attr{Key: k, Value: slog.AnyValue(v)}
		slogAttrs = append(slogAttrs, slogAttr)
	}

	w.log.Handler().WithAttrs(slogAttrs)

	//REFS: the WithGroup method in slog.Handler not interact with the log
	// directly, so we need to write the group name to the buffer manually
	if w.params.GetBuffered() {
		w.buf.WriteString(w.mapToString(attrs))
	}

	return w
}

func (w *WsLogger) WithGroup(groupName string) *WsLogger {
	//REFS: the WithGroup method in slog.Handler not interact with the log
	// directly, so we need to write the group name to the buffer manually
	if w.params.GetBuffered() {
		w.buf.WriteString(groupName)
	}
	w.log = w.log.WithGroup(groupName)
	return w
}

func (w *WsLogger) GetBuf() string {
	return w.buf.String()
}

func (w *WsLogger) ResetBuf() {
	w.buf.Reset()
}

func (w *WsLogger) mapToString(attrs map[string]interface{}) string {
	var buf bytes.Buffer
	for k, v := range attrs {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(v.(string))
		buf.WriteString(" ")
	}
	return buf.String()
}
