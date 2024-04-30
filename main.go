package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	dsync "github.com/adiom-data/dsync/app"
)

func main() {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGPIPE)
	cancellableCtx, cancelApp := context.WithCancel(context.Background())

	go func() {
		for s := range sigChan {
			if s != syscall.SIGPIPE {
				cancelApp()
				break
			}
		}
	}()

	app := dsync.NewApp()
	err := app.RunContext(cancellableCtx, os.Args)
	if err != nil {
		slog.Error(err.Error())
	}
}
