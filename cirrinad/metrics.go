package main

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cirrina/cirrinad/config"
	"cirrina/cirrinad/vm"
)

func serveMetrics() {
	if !config.Config.Metrics.Enabled {
		return
	}

	metricsAddr := net.JoinHostPort(config.Config.Metrics.Host, strconv.FormatUint(uint64(config.Config.Metrics.Port), 10))

	slog.Debug("serving metrics", "metricsAddr", metricsAddr)

	http.Handle("/metrics", promhttp.Handler())

	// Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
	err := http.ListenAndServe(metricsAddr, nil) //nolint:gosec
	if err != nil {
		slog.Error("error serving metrics", "err", err)
	}
}

func setupMetrics() {
	vm.SetupVMMetrics()
}
