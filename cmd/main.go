package main

import (
	"context"
	"github.com/alexpain/barbuddy/app"
	"github.com/alexpain/barbuddy/config"
	"golang.org/x/sync/errgroup"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic("failed to parse the configuration: " + err.Error())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	a, err := app.New(ctx, cfg)
	if err != nil {
		panic("failed to initiate the application: " + err.Error())
	}

	err = a.Run(ctx)
	if err != nil {
		return
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-gCtx.Done()
		stop()

		log.Print("graceful shutdown")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
		defer cancel()

		return a.Stop(ctx)
	})

	if err := g.Wait(); err != nil {
		log.Panic("something went wrong", slog.String("error", err.Error()))
	}
}
