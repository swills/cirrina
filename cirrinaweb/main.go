package main

import (
	"embed"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
	"github.com/spf13/cast"

	"cirrina/cirrinaweb/handlers"
	"cirrina/cirrinaweb/util"
)

//go:generate go run github.com/a-h/templ/cmd/templ generate

var metricsEnable bool
var mdlw middleware.Middleware

//go:embed assets/*
var assetFS embed.FS

func healthCheck(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusNoContent)
}

func parseEnv() (string, uint16) {
	var err error

	util.InitRPC(
		os.Getenv("CIRRINAWEB_CIRRINAHOST"),
		os.Getenv("CIRRINAWEB_CIRRINAPORT"),
		os.Getenv("CIRRINAWEB_CIRRINATIMEOUT"),
	)

	util.SetListenHost(os.Getenv("CIRRINAWEB_HOST"))

	util.SetListenPort(os.Getenv("CIRRINAWEB_PORT"))

	enableMetricsStr := os.Getenv("CIRRINAWEB_METRICS_ENABLE")
	if enableMetricsStr == "true" {
		metricsEnable = true
	}

	metricsHost := os.Getenv("CIRRINAWEB_METRICS_HOST")
	if metricsHost == "" {
		metricsHost = "localhost"
	}

	var metricsPort uint16 = 9090

	metricsPortStr := os.Getenv("CIRRINAWEB_METRICS_PORT")
	if metricsPortStr != "" {
		var metricsPort64 uint64

		metricsPort64, err = strconv.ParseUint(metricsPortStr, 10, 16)
		if err == nil {
			metricsPort = cast.ToUint16(metricsPort64)
		}
	}

	util.SetWebsockifyPort(os.Getenv("CIRRINAWEB_WEBSOCKIFYPORT"))

	util.SetAccessLog(os.Getenv("CIRRINAWEB_ACCESSLOG"))
	util.SetErrorLog(os.Getenv("CIRRINAWEB_ERRORLOG"))

	return metricsHost, metricsPort
}

func setupMetrics(host string, port uint16) {
	go func() {
		srv := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Addr:         net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)),
			Handler:      promhttp.Handler(),
		}

		err := srv.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
}

//nolint:funlen
func main() {
	var err error

	var metricsHost string

	var metricsPort uint16

	metricsHost, metricsPort = parseEnv()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthCheck)
	mux.HandleFunc("GET /favicon.ico", handlers.FaviconHandlerFunc)

	vncFileServer := http.FileServerFS(vncFS)
	assetFileServer := http.FileServerFS(assetFS)

	if metricsEnable {
		mdlw = middleware.New(middleware.Config{
			Recorder: metrics.NewRecorder(metrics.Config{}),
		})

		mux.Handle("GET /", HTTPLogger(middlewarestd.Handler("/", mdlw, handlers.NewHomeHandler())))
		mux.Handle("GET /home", HTTPLogger(middlewarestd.Handler("/home", mdlw, handlers.NewHomeHandler())))
		mux.Handle("GET /vms", HTTPLogger(middlewarestd.Handler("/vms", mdlw, handlers.NewVMsHandler())))
		mux.Handle("GET /vm/{nameOrID}", HTTPLogger(middlewarestd.Handler("/vm/:nameOrID", mdlw, handlers.NewVMHandler())))
		mux.Handle("DELETE /vm/{nameOrID}", HTTPLogger(middlewarestd.Handler("/vm/:nameOrID", mdlw, handlers.NewVMHandler())))
		mux.Handle(
			"POST /vm/{nameOrID}/start",
			HTTPLogger(middlewarestd.Handler("/vm/{nameOrID}/start", mdlw, handlers.NewVMStartHandler())),
		)
		mux.Handle(
			"POST /vm/{nameOrID}/stop",
			HTTPLogger(middlewarestd.Handler("/vm/{nameOrID}/stop", mdlw, handlers.NewVMStopHandler())),
		)
		mux.Handle(
			"DELETE /vm/{vmNameOrID}/disk/{diskNameOrID}",
			HTTPLogger(middlewarestd.Handler("/vm/{vmNameOrID}/disk/{diskNameOrID}", mdlw, handlers.NewVMDiskHandler())),
		)
		mux.Handle(
			"DELETE /vm/{vmNameOrID}/iso/{isoNameOrID}",
			HTTPLogger(middlewarestd.Handler("/vm/{vmNameOrID}/iso/{isoNameOrID}", mdlw, handlers.NewVMISOHandler())),
		)
		mux.Handle(
			"DELETE /vm/{vmNameOrID}/nic/{nicNameOrID}",
			HTTPLogger(middlewarestd.Handler("/vm/{vmNameOrID}/nic/{isoNameOrID}", mdlw, handlers.NewVMNICHandler())),
		)

		mux.Handle(
			"GET /vmdata/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/vm/:nameOrID", mdlw, handlers.NewVMDataHandler())),
		)
		mux.Handle("GET /vnc/", HTTPLogger(middlewarestd.Handler("/vnc/", mdlw, NoCache(vncFileServer))))

		mux.Handle("GET /media/disks", HTTPLogger(middlewarestd.Handler("/media/disks", mdlw, handlers.NewDisksHandler())))
		mux.Handle(
			"GET /media/disk/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/disks/:nameOrID", mdlw, handlers.NewDiskHandler())),
		)
		mux.Handle(
			"DELETE /media/disk/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/disk/:nameOrID", mdlw, handlers.NewDiskHandler())),
		)

		mux.Handle("GET /media/isos", HTTPLogger(middlewarestd.Handler("/media/isos", mdlw, handlers.NewISOsHandler())))
		mux.Handle(
			"GET /media/iso/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/isos/:nameOrID", mdlw, handlers.NewISOHandler())),
		)
		mux.Handle(
			"DELETE /media/iso/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/iso/:nameOrID", mdlw, handlers.NewISOHandler())),
		)

		mux.Handle("GET /net/nics", HTTPLogger(middlewarestd.Handler("/net/nics", mdlw, handlers.NewNICsHandler())))
		mux.Handle(
			"GET /net/nic/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/net/nic/:nameOrID", mdlw, handlers.NewNICHandler())),
		)
		mux.Handle("DELETE /net/nic/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/net/nic/:nameOrID", mdlw, handlers.NewNICHandler())),
		)

		mux.Handle(
			"GET /net/switches",
			HTTPLogger(middlewarestd.Handler("/net/switches", mdlw, handlers.NewSwitchesHandler())))
		mux.Handle(
			"GET /net/switch/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/net/switch/:nameOrID", mdlw, handlers.NewSwitchHandler())),
		)
		mux.Handle(
			"DELETE /net/switch/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/net/switch/:nameOrID", mdlw, handlers.NewSwitchHandler())),
		)

		mux.Handle("GET /assets/", HTTPLogger(middlewarestd.Handler("/assets/", mdlw, assetFileServer)))

		setupMetrics(metricsHost, metricsPort)
	} else {
		mux.Handle("GET /", HTTPLogger(handlers.NewHomeHandler()))
		mux.Handle("GET /home", HTTPLogger(handlers.NewHomeHandler()))
		mux.Handle("GET /vms", HTTPLogger(handlers.NewVMsHandler()))
		mux.Handle("GET /vm/{nameOrID}", HTTPLogger(handlers.NewVMHandler()))
		mux.Handle("DELETE /vm/{nameOrID}", HTTPLogger(handlers.NewVMHandler()))
		mux.Handle("POST /vm/{nameOrID}/start", HTTPLogger(handlers.NewVMStartHandler()))
		mux.Handle("POST /vm/{nameOrID}/stop", HTTPLogger(handlers.NewVMStopHandler()))
		mux.Handle("DELETE /vm/{vmNameOrID}/disk/{diskNameOrID}", HTTPLogger(handlers.NewVMDiskHandler()))
		mux.Handle("DELETE /vm/{vmNameOrID}/iso/{isoNameOrID}", HTTPLogger(handlers.NewVMISOHandler()))
		mux.Handle("DELETE /vm/{vmNameOrID}/nic/{isoNameOrID}", HTTPLogger(handlers.NewVMNICHandler()))
		mux.Handle("GET /vmdata/{nameOrID}", HTTPLogger(handlers.NewVMDataHandler()))
		mux.Handle("GET /vnc/", HTTPLogger(NoCache(vncFileServer)))
		mux.Handle("GET /assets/", HTTPLogger(assetFileServer))
		mux.Handle("GET /media/disks", HTTPLogger(handlers.NewDisksHandler()))
		mux.Handle("DELETE /media/disk/{nameOrID}", HTTPLogger(handlers.NewDiskHandler()))
		mux.Handle("GET /media/disk/{nameOrID}", HTTPLogger(handlers.NewDiskHandler()))
		mux.Handle("GET /media/isos", HTTPLogger(handlers.NewISOsHandler()))
		mux.Handle("GET /media/iso/{nameOrID}", HTTPLogger(handlers.NewISOHandler()))
		mux.Handle("DELETE /media/iso/{nameOrID}", HTTPLogger(handlers.NewISOHandler()))
		mux.Handle("GET /net/nics", HTTPLogger(handlers.NewNICsHandler()))
		mux.Handle("GET /net/nic/{nameOrID}", HTTPLogger(handlers.NewNICHandler()))
		mux.Handle("DELETE /net/nic/{nameOrID}", HTTPLogger(handlers.NewNICHandler()))
		mux.Handle("GET /net/switches", HTTPLogger(handlers.NewSwitchesHandler()))
		mux.Handle("GET /net/switch/{nameOrID}", HTTPLogger(handlers.NewSwitchHandler()))
		mux.Handle("DELETE /net/switch/{nameOrID}", HTTPLogger(handlers.NewSwitchHandler()))
	}

	go StartGoWebSockifyHTTP()

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         net.JoinHostPort(util.GetListenHost(), strconv.FormatUint(uint64(util.GetListenPort()), 10)),
		Handler:      mux,
	}

	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
