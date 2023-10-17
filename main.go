package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
	"traefik-http-lxd-provider/client"
	"traefik-http-lxd-provider/worker"

	"github.com/gorilla/mux"
)

func main() {
	w := worker.NewWorkerPool(10)
	lxd := client.NewClientConnectionPool(client.PoolConfig{
		MaxPoolSize:           10,
		MaxIdleConnections:    8,
		IdleConnectionTimeout: 1 * time.Minute,
	})

	instanceManager, err := NewInstanceManager(w, lxd)
	if err != nil {
		panic(fmt.Sprintf("error creating instance manager, err: %v", err))
	}

	handler := HTTPHandler{
		im: instanceManager,
	}

	r := mux.NewRouter()
	r.HandleFunc("/services/http", handler.ProvideHTTPServices)
	r.HandleFunc("/services/tcp", handler.ProvideTCPServices)

	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			slog.Error("error listen server", "error", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		os.Exit(0)
	}
	slog.Info("shutting down")
	os.Exit(0)
}
