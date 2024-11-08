package main

import (
	"context"
	"github.com/alexpain/barbuddy/internal/app"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	_, err := app.New(ctx)
	if err != nil {
		return
	}
}
