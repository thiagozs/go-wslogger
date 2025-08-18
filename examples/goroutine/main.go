package main

import (
	"os"
	"sync"
	"time"

	logger "github.com/thiagozs/go-wslogger"
)

// Helper para criar goroutine capturando local de criação
func GoRoutineCall(wg *sync.WaitGroup, log *logger.Logger) {
	// usa o wrapper do logger para capturar o ponto de criação
	//g := log.WrapGoroutine()
	go func(wg *sync.WaitGroup, lg *logger.Logger) {
		defer wg.Done()
		log.Info("executando GoRoutineCall <> dentro da goroutine")
	}(wg, log)
}

func GoRoutineCall2(wg *sync.WaitGroup, log *logger.Logger) {
	go func(wg *sync.WaitGroup, lg *logger.Logger) {
		defer wg.Done()
		log.Info("executando GoRoutineCall2 :: dentro da goroutine")
	}(wg, log)
}

func main() {
	log := logger.NewLogger(
		logger.WithColor(false),
		logger.WithWriter(os.Stdout),
	)
	var wg sync.WaitGroup

	wg.Add(3)
	for range 3 {
		GoRoutineCall(&wg, log)
	}

	wg.Add(3)
	for range 3 {
		GoRoutineCall2(&wg, log)
	}

	wg.Wait()

	// Log normal para comparação
	log.Info("log chamado na main", "goroutine_caller", "main.go:main")

	// Espera para garantir saída das goroutines
	time.Sleep(200 * time.Millisecond)
}
