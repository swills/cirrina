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

func parseEnv() (string, uint64) {
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

	var metricsPort uint64 = 9090

	metricsPortStr := os.Getenv("CIRRINAWEB_METRICS_PORT")
	if metricsPortStr != "" {
		metricsPort, err = strconv.ParseUint(metricsPortStr, 10, 64)
		if err != nil || metricsPort > 65536 {
			metricsPort = 9090
		}
	}

	util.SetWebsockifyPort(os.Getenv("CIRRINAWEB_WEBSOCKIFYPORT"))

	util.SetAccessLog(os.Getenv("CIRRINAWEB_ACCESSLOG"))
	util.SetErrorLog(os.Getenv("CIRRINAWEB_ERRORLOG"))

	return metricsHost, metricsPort
}

func setupMetrics(host string, port uint64) {
	go func() {
		srv := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Addr:         net.JoinHostPort(host, strconv.FormatUint(port, 10)),
			Handler:      promhttp.Handler(),
		}

		err := srv.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
}

func main() {
	var err error

	var metricsHost string

	var metricsPort uint64

	metricsHost, metricsPort = parseEnv()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthCheck)
	mux.HandleFunc("GET /favicon.ico", handlers.FaviconHandlerFunc)

	vncFileServer := http.FileServer(http.FS(vncFS))
	assetFileServer := http.FileServer(http.FS(assetFS))

	if metricsEnable {
		mdlw = middleware.New(middleware.Config{
			Recorder: metrics.NewRecorder(metrics.Config{}),
		})

		mux.Handle("GET /", HTTPLogger(middlewarestd.Handler("/", mdlw, handlers.NewHomeHandler())))
		mux.Handle("GET /home", HTTPLogger(middlewarestd.Handler("/home", mdlw, handlers.NewHomeHandler())))
		mux.Handle("GET /vms", HTTPLogger(middlewarestd.Handler("/vms", mdlw, handlers.NewVMsHandler())))
		mux.Handle("GET /vm/{nameOrID}", HTTPLogger(middlewarestd.Handler("/vm/:nameOrID", mdlw, handlers.NewVMHandler())))
		mux.Handle(
			"POST /vm/{nameOrID}/start",
			HTTPLogger(middlewarestd.Handler("/vm/{nameOrID}/start", mdlw, handlers.NewVMStartHandler())),
		)
		mux.Handle(
			"POST /vm/{nameOrID}/stop",
			HTTPLogger(middlewarestd.Handler("/vm/{nameOrID}/stop", mdlw, handlers.NewVMStopHandler())),
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

		mux.Handle("GET /media/isos", HTTPLogger(middlewarestd.Handler("/media/isos", mdlw, handlers.NewISOsHandler())))
		mux.Handle(
			"GET /media/iso/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/isos/:nameOrID", mdlw, handlers.NewISOHandler())),
		)

		mux.Handle("GET /net/nics", HTTPLogger(middlewarestd.Handler("/media/isos", mdlw, handlers.NewNICsHandler())))
		mux.Handle(
			"GET /net/nic/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/isos/:nameOrID", mdlw, handlers.NewNICHandler())),
		)

		mux.Handle("GET /net/switches", HTTPLogger(middlewarestd.Handler("/media/isos", mdlw, handlers.NewSwitchesHandler())))
		mux.Handle(
			"GET /net/switch/{nameOrID}",
			HTTPLogger(middlewarestd.Handler("/media/isos/:nameOrID", mdlw, handlers.NewSwitchHandler())),
		)

		mux.Handle("GET /assets/", HTTPLogger(middlewarestd.Handler("/assets/", mdlw, assetFileServer)))

		setupMetrics(metricsHost, metricsPort)
	} else {
		mux.Handle("GET /", HTTPLogger(handlers.NewHomeHandler()))
		mux.Handle("GET /home", HTTPLogger(handlers.NewHomeHandler()))
		mux.Handle("GET /vms", HTTPLogger(handlers.NewVMsHandler()))
		mux.Handle("GET /vm/{nameOrID}", HTTPLogger(handlers.NewVMHandler()))
		mux.Handle("POST /vm/{nameOrID}/start", HTTPLogger(handlers.NewVMStartHandler()))
		mux.Handle("POST /vm/{nameOrID}/stop", HTTPLogger(handlers.NewVMStopHandler()))
		mux.Handle("GET /vmdata/{nameOrID}", HTTPLogger(handlers.NewVMDataHandler()))
		mux.Handle("GET /vnc/", HTTPLogger(NoCache(vncFileServer)))
		mux.Handle("GET /assets/", HTTPLogger(assetFileServer))
		mux.Handle("GET /media/disks", HTTPLogger(handlers.NewDisksHandler()))
		mux.Handle("GET /media/disk/{nameOrID}", HTTPLogger(handlers.NewDiskHandler()))
		mux.Handle("GET /media/isos", HTTPLogger(handlers.NewISOsHandler()))
		mux.Handle("GET /media/iso/{nameOrID}", HTTPLogger(handlers.NewISOHandler()))
		mux.Handle("GET /net/nics", HTTPLogger(handlers.NewNICsHandler()))
		mux.Handle("GET /net/nic/{nameOrID}", HTTPLogger(handlers.NewNICHandler()))
		mux.Handle("GET /net/switches", HTTPLogger(handlers.NewSwitchesHandler()))
		mux.Handle("GET /net/switch/{nameOrID}", HTTPLogger(handlers.NewSwitchHandler()))
	}

	go StartGoWebSockifyHTTP()

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         net.JoinHostPort(util.GetListenHost(), strconv.FormatUint(util.GetListenPort(), 10)),
		Handler:      mux,
	}

	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
