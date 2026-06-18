//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fdcs99/biligo/internal/app"
)

func run(configPath string) error {
	service := app.New(configPath)
	if err := service.Start(context.Background()); err != nil {
		return err
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return service.Stop(ctx)
}
