package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	file, err := os.OpenFile("linko.access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("Could not open log file: ", err)
	}
	st, err := store.New(dataDir, log.New(os.Stderr, "DEBUG: ", log.LstdFlags))
	if err != nil {
		return 1
	}
	s := newServer(*st, httpPort, cancel, log.New(file, "INFO: ", log.LstdFlags))
	var serverErr error
	go func() {
		serverErr = s.start()
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.shutdown(shutdownCtx); err != nil {
		return 1
	}
	if serverErr != nil {
		return 1
	}
	return 0
}
