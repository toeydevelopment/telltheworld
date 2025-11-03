package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/toeydevelopment/telltheworld/apps/teller/internal/conf"

	_ "time/tzdata"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name = "teller-worker"
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "./apps/teller/config/config.yaml", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
			env.NewSource(),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// Wire the app
	app, cleanup, err := wireWorker(bc.Data, bc.Pubsub, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in goroutine
	go func() {
		log.NewHelper(logger).Info("Starting notification worker...")
		if err := app.Start(ctx); err != nil {
			log.NewHelper(logger).Errorf("Worker error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.NewHelper(logger).Infof("Received signal: %v", sig)

	// Cancel context to trigger graceful shutdown
	cancel()

	// Stop the worker
	if err := app.Stop(); err != nil {
		log.NewHelper(logger).Errorf("Failed to stop worker: %v", err)
	}

	log.NewHelper(logger).Info("Worker shutdown complete")
}