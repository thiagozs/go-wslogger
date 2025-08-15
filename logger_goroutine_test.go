package wslogger

import (
	"sync"
	"testing"
)

func TestLogger_GoroutineID(t *testing.T) {
	l := NewLogger(WithFormat("[{caller}] {message}"), WithColor(false))
	var wg sync.WaitGroup
	results := make(chan string, 5)

	logInGoroutine := func(_ int) {
		defer wg.Done()
		msg := l.getCaller(2)
		results <- msg
	}

	wg.Add(5)
	for i := range 5 {
		go logInGoroutine(i)
	}
	wg.Wait()
	close(results)

	ids := make(map[string]bool)
	for res := range results {
		t.Logf("caller: %s", res)
		if res == "unknown" {
			t.Errorf("getCaller returned unknown")
		}
		ids[res] = true
	}
	if len(ids) < 2 {
		t.Errorf("Expected at least 2 unique goroutine IDs, got %d", len(ids))
	}
}
