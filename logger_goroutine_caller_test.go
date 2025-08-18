package wslogger

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// Testa o fluxo do examples/goroutine: chama WrapGoroutine no criador, usa o wrapper
// dentro da goroutine e valida que a saída JSON contém o campo goroutine_caller.
func TestLogger_GoroutineFlow(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(WithWriter(&buf), WithJSON(true), WithColor(false))

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		// captura callsite no ponto de criação
		g := log.WrapGoroutine()
		go func(g *GoroutineLogger) {
			defer wg.Done()
			g.Info("hello from goroutine")
		}(g)
	}

	// espera com timeout para evitar flakiness
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for goroutines")
	}

	// dar um pequeno intervalo para garantir flush (writer é buffer direto, mas por segurança)
	time.Sleep(10 * time.Millisecond)

	out := buf.String()
	if out == "" {
		t.Fatal("no log output captured")
	}

	lines := bytes.Split([]byte(out), []byte("\n"))
	var nonEmpty [][]byte
	for _, l := range lines {
		if len(l) > 0 {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) < n {
		t.Fatalf("expected at least %d log lines, got %d", n, len(nonEmpty))
	}

	type record struct {
		Extra map[string]string `json:"extra"`
	}

	for i := 0; i < n; i++ {
		var r record
		if err := json.Unmarshal(nonEmpty[i], &r); err != nil {
			t.Fatalf("invalid json log line: %v, line=%s", err, string(nonEmpty[i]))
		}
		if r.Extra == nil {
			t.Fatalf("missing extra in log line: %s", string(nonEmpty[i]))
		}
		if v, ok := r.Extra["goroutine_caller"]; !ok || v == "" {
			t.Fatalf("goroutine_caller missing or empty in extra: %v", r.Extra)
		}
	}
}
