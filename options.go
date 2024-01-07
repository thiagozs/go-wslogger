package wslogger

type Options func(*WsLoggerParams) error

type KindLogger int

const (
	Default KindLogger = iota
	Stdout
	File
	JSON
)

func (k KindLogger) String() string {
	return [...]string{"default", "stdout", "file"}[k]
}

type WsLoggerParams struct {
	kindLogger KindLogger
	fileName   string
	buffered   bool
}

func newWsLoggerParams(opts ...Options) (*WsLoggerParams, error) {
	params := &WsLoggerParams{}

	for _, opt := range opts {
		if err := opt(params); err != nil {
			return nil, err
		}
	}

	return params, nil
}

func WithKind(kind KindLogger) Options {
	return func(params *WsLoggerParams) error {
		params.kindLogger = kind
		return nil
	}
}

func WithFileName(filename string) Options {
	return func(params *WsLoggerParams) error {
		params.fileName = filename
		return nil
	}
}

func WithBuffered() Options {
	return func(params *WsLoggerParams) error {
		params.buffered = true
		return nil
	}
}

// getters -----

func (p *WsLoggerParams) GetKind() KindLogger {
	return p.kindLogger
}

func (p *WsLoggerParams) GetFileName() string {
	return p.fileName
}

func (p *WsLoggerParams) GetBuffered() bool {
	return p.buffered
}

// setters -----

func (p *WsLoggerParams) SetKind(kind KindLogger) {
	p.kindLogger = kind
}

func (p *WsLoggerParams) SetFileName(filename string) {
	p.fileName = filename
}

func (p *WsLoggerParams) SetBuffered(buffered bool) {
	p.buffered = buffered
}
