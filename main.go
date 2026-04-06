package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"log/slog"
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

type closeFunc func() error

func initializeLogger() (*slog.Logger, closeFunc, error) {
	var buffionWriter *bufio.Writer
	var multiWriter io.Writer = os.Stderr

	logFileName, exists := os.LookupEnv("LINKO_LOG_FILE")
	if exists {
		file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return nil, nil, err // Return the error instead of crashing
		}
		buffionWriter = bufio.NewWriterSize(file, 8192)
		multiWriter = io.MultiWriter(os.Stderr, buffionWriter)
	}
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// slog requires a infoHandler to format the output
	infoHandler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(slog.NewMultiHandler(
		debugHandler,
		infoHandler,
	))

	cleanup := func() error {
		if buffionWriter != nil {
			return buffionWriter.Flush()
		}
		return nil
	}

	return logger, cleanup, nil
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	logger, closeLogger, err := initializeLogger()
	if err != nil {
		return 1
	}
	defer func() {
		if err := closeLogger(); err != nil {
			logger.Error("failed to close logger", slog.Any("error", err))
		}
	}()

	st, err := store.New(dataDir, logger)
	if err != nil {
		return 1
	}
	s := newServer(*st, httpPort, cancel, logger)
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
