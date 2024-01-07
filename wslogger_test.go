package wslogger

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	opts = []Options{
		WithFileName("/tmp/test.log"),
		WithBuffered(),
	}
)

func TestNewWsLoggerDefault(t *testing.T) {
	opts = append(opts, WithKind(Default))

	lg, err := NewWsLogger(opts...)
	assert.Equal(t, err, nil)

	assert.Equal(t, lg.params.GetKind(), Default)

}

func TestNewWsLoggerStdout(t *testing.T) {
	optsStdout := []Options{WithKind(Stdout)}

	lg, err := NewWsLogger(optsStdout...)
	assert.Equal(t, err, nil)

	assert.Equal(t, lg.params.GetKind(), Stdout)
}

func TestNewWsLoggerFile(t *testing.T) {
	optsFile := []Options{WithKind(File), WithFileName("/tmp/test.log")}

	lg, err := NewWsLogger(optsFile...)
	assert.Equal(t, err, nil)

	assert.Equal(t, lg.params.GetKind(), File)
}

func TestNewWsLoggerJSON(t *testing.T) {
	optsJSON := []Options{WithKind(JSON)}

	lg, err := NewWsLogger(optsJSON...)
	assert.Equal(t, err, nil)

	assert.Equal(t, lg.params.GetKind(), JSON)
}

func TestWsLoggerWithGroup(t *testing.T) {
	logger, err := NewWsLogger(opts...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	groupName := "TestGroup"
	logger.WithGroup(groupName).Info("TestMessage")

	if !strings.Contains(logger.GetBuf(), groupName) {
		t.Errorf("Expected buffer to contain group name '%s'", groupName)
	}
}

func TestWsLoggerInfo(t *testing.T) {
	logger, err := NewWsLogger(opts...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	msg := "TestMessage"
	logger.Info(msg)

	if !strings.Contains(logger.GetBuf(), msg) {
		t.Errorf("Expected buffer to contain message '%s'", msg)
	}
}

func TestWsLoggerDebug(t *testing.T) {
	logger, err := NewWsLogger(opts...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	msg := "TestMessage"
	logger.Debug(msg)

	if !strings.Contains(logger.GetBuf(), msg) {
		t.Errorf("Expected buffer to contain message '%s'", msg)
	}
}

func TestWsLoggerError(t *testing.T) {
	logger, err := NewWsLogger(opts...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	msg := "TestMessage"
	logger.Error(msg)

	if !strings.Contains(logger.GetBuf(), msg) {
		t.Errorf("Expected buffer to contain message '%s'", msg)
	}
}

func TestWsLoggerWarn(t *testing.T) {
	logger, err := NewWsLogger(opts...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	msg := "TestMessage"
	logger.Warn(msg)

	if !strings.Contains(logger.GetBuf(), msg) {
		t.Errorf("Expected buffer to contain message '%s'", msg)
	}
}

func TestWsLoggerWithAttrs(t *testing.T) {
	logger, err := NewWsLogger(opts...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	attrs := map[string]interface{}{
		"TestKey": "TestValue",
	}

	logger.WithAttrs(attrs).Info("TestMessage")

	if !strings.Contains(logger.GetBuf(), attrs["TestKey"].(string)) {
		t.Errorf("Expected buffer to contain attribute '%s'", attrs["TestKey"].(string))
	}
}
