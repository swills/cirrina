package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cirrina/cirrinad/vm"
)

func newMetricsServer(serverAddr string, mux *http.ServeMux) *http.Server {
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         serverAddr,
		Handler:      mux,
	}
}

func serveMetrics(metricsAddr string) {
	slog.Debug("serving metrics", "metricsAddr", metricsAddr)

	mux := http.NewServeMux()

	srv := newMetricsServer(metricsAddr, mux)

	mux.Handle("GET /metrics", promhttp.Handler())

	err := srv.ListenAndServe()
	if err != nil {
		slog.Error("error serving metrics", "err", err)
	}
}

func setupMetrics() {
	vm.SetupVMMetrics()
}
