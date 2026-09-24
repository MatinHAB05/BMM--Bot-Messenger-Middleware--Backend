// Command app is the messenger-backend entrypoint: it assembles the
// application via bootstrap.Init, starts the Telegram/Bale background bot
// update listeners, serves the HTTP API, and shuts everything down
// gracefully on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"messenger-backend/bootstrap"
	"messenger-backend/pkg/logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.Init(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Background bot update listeners run for the lifetime of ctx and are
	// torn down automatically when it's cancelled below.
	app.StartListeners(ctx)

	server := &http.Server{
		Addr:         ":" + app.Env.App.Port,
		Handler:      app.Router,
		ReadTimeout:  app.Const.Server.HTTPReadTimeout,
		WriteTimeout: app.Const.Server.HTTPWriteTimeout,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		app.Logger.Info("http server listening", logger.String("port", app.Env.App.Port), logger.String("env", app.Env.App.AppEnv))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		app.Logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		app.Logger.Error(err, "http server error, shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.Const.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error(err, "error during http server shutdown")
	}

	app.Shutdown(shutdownCtx)
	app.Logger.Info("application stopped")
}
