package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexisvisco/koyebtests/internal/handler"
	"github.com/alexisvisco/koyebtests/internal/service"
	"github.com/hashicorp/nomad/api"
)

var (
	host    = "koyebtest.alexisvis.co"
	apiHost = "api.koyebtest.alexisvis.co"
)

func main() {
	logger := slog.With("component", "main")

	if os.Getenv("HOST") != "" {
		host = os.Getenv("HOST")
	}

	if os.Getenv("API_HOST") != "" {
		apiHost = os.Getenv("API_HOST")
	}

	config := api.DefaultConfig()

	nomadClient, err := api.NewClient(config)
	if err != nil {
		logger.Error("unable to create Nomad client", "error", err)
		os.Exit(1)
	}

	_, err = nomadClient.Agent().Self()
	if err != nil {
		logger.Error("unable to connect to Nomad", "error", err)
	} else {
		logger.Info("successfully connected to Nomad", "address", nomadClient.Address())
	}

	jobService := service.NewNomadJobService(host, nomadClient)

	mainHandler := handler.Main(handler.MainParams{
		Host:       host,
		ApiHost:    apiHost,
		JobService: jobService,
	})

	http.HandleFunc("PUT /services/:name", handler.CreateJob(jobService))

	server := &http.Server{
		Addr:    ":80",
		Handler: mainHandler,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "port", server.Addr)
		logger.Info("API endpoint", "host", apiHost)
		logger.Info("subdomain reverse proxy", "pattern", "*."+host, "target", "localhost")

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	if err := jobService.Close(); err != nil {
		logger.Error("error closing jobs", "error", err)
		os.Exit(1)
	}

	logger.Info("server exited gracefully")
}
