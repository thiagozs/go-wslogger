package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"runtime"

	logger "github.com/thiagozs/go-wslogger"
)

// Helper para criar goroutine capturando local de criação
func GoWithCaller(fn func(file string, line int)) {
	_, file, line, _ := runtime.Caller(1)
	go fn(file, line)
}

func main() {
	log := logger.NewLogger(
		logger.WithFormat("{caller} {message} {extra}"),
		logger.WithColor(false),
		logger.WithWriter(os.Stdout),
	)
	var wg sync.WaitGroup

	logInGoroutine := func(file string, line int) {
		defer wg.Done()
		// Passa o local de criação como extra para o logger
		log.Info("executando dentro da goroutine", "goroutine_caller", fmt.Sprintf("%s:%d", file, line))
	}

	wg.Add(3)
	for i := 0; i < 3; i++ {
		GoWithCaller(logInGoroutine)
	}
	wg.Wait()

	// Log normal para comparação
	log.Info("log chamado na main", "goroutine_caller", "main.go:main")

	// Espera para garantir saída das goroutines
	time.Sleep(200 * time.Millisecond)
}
