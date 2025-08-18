package wslogger

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// Helper para criar goroutine capturando local de criação
func GoWithCaller(fn func(file string, line int)) {
	_, file, line, _ := runtime.Caller(1)
	go fn(file, line)
}

func TestLogger_GoroutineCreationCaller(t *testing.T) {
	var wg sync.WaitGroup
	results := make(chan string, 5)

	logInGoroutine := func(file string, line int) {
		defer wg.Done()
		msg := fmt.Sprintf("goroutine criada em %s:%d", file, line)
		results <- msg
	}

	wg.Add(5)
	for i := 0; i < 5; i++ {
		GoWithCaller(logInGoroutine)
	}
	wg.Wait()
	close(results)

	for res := range results {
		t.Logf(res)
		if res == "" {
			t.Errorf("caller info vazia")
		}
	}
}
