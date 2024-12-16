package main

import (
	"embed"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
)

//go:generate go run github.com/a-h/templ/cmd/templ generate

//go:embed favicon.ico
var favicon []byte

var cirrinaServerName string
var cirrinaServerPort uint16
var cirrinaServerTimeout uint64
var listenHost string
var listenPort uint64
var websockifyPort uint16
var metricsEnable bool
var mdlw middleware.Middleware
var accessLog *os.File
var errorLog *os.File

//go:embed assets/*
var assetFS embed.FS

func healthCheck(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusNoContent)
}

func faviconHandlerFunc(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(favicon)
}

func homeHandlerFunc(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/" || request.URL.Path == "/home" {
		templHandler := HTTPLogger(templ.Handler(home()))

		if metricsEnable {
			middlewarestd.Handler(request.URL.Path, mdlw, templHandler).ServeHTTP(writer, request)
		} else {
			templHandler.ServeHTTP(writer, request)
		}

		return
	}

	notFoundHandler := HTTPLogger(templ.Handler(notFoundComponent(), templ.WithStatus(http.StatusNotFound)))

	notFoundHandler.ServeHTTP(writer, request)
}

func parseEnv() (string, uint64) { //nolint:funlen,cyclop
	var err error

	cirrinaServerName = os.Getenv("CIRRINAWEB_CIRRINAHOST")
	if cirrinaServerName == "" {
		cirrinaServerName = "localhost"
	}

	cirrinaServerPortString := os.Getenv("CIRRINAWEB_CIRRINAPORT")
	if cirrinaServerPortString == "" {
		cirrinaServerPort = 50051
	} else {
		var cirrinaServerPortTemp uint64

		cirrinaServerPortTemp, err = strconv.ParseUint(cirrinaServerPortString, 10, 16)
		if err != nil {
			cirrinaServerPort = 50051
		} else {
			cirrinaServerPort = uint16(cirrinaServerPortTemp)
		}
	}

	cirrinaServerTimeoutString := os.Getenv("CIRRINAWEB_CIRRINATIMEOUT")
	if cirrinaServerTimeoutString == "" {
		cirrinaServerTimeout = 5
	} else {
		cirrinaServerTimeout, err = strconv.ParseUint(cirrinaServerTimeoutString, 10, 64)
		if err != nil {
			cirrinaServerTimeout = 5
		}
	}

	listenHost = os.Getenv("CIRRINAWEB_HOST")
	if listenHost == "" {
		listenHost = "localhost"
	}

	listenPortString := os.Getenv("CIRRINAWEB_PORT")
	if listenPortString != "" {
		listenPort, err = strconv.ParseUint(listenPortString, 10, 16)
		if err != nil || listenPort > 65536 {
			listenPort = 8888
		}
	} else {
		listenPort = 8888
	}

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

	cirrinaWebsockifyPortString := os.Getenv("CIRRINAWEB_WEBSOCKIFYPORT")
	if cirrinaWebsockifyPortString == "" {
		websockifyPort = 7900
	} else {
		var cirrinaWebsockifyPortTemp uint64

		cirrinaWebsockifyPortTemp, err = strconv.ParseUint(cirrinaServerPortString, 10, 16)
		if err != nil {
			websockifyPort = 7900
		} else {
			websockifyPort = uint16(cirrinaWebsockifyPortTemp)
		}
	}

	accessLogFile := os.Getenv("CIRRINAWEB_ACCESSLOG")
	if accessLogFile == "" {
		accessLog = os.Stdout
	} else {
		accessLog, err = os.OpenFile(accessLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			accessLog = os.Stdout
		}
	}

	errorLogFile := os.Getenv("CIRRINAWEB_ERRORLOG")
	if errorLogFile == "" {
		errorLog = os.Stdout
	} else {
		errorLog, err = os.OpenFile(errorLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			errorLog = os.Stderr
		}
	}

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
	mux.HandleFunc("GET /favicon.ico", faviconHandlerFunc)
	mux.HandleFunc("GET /", homeHandlerFunc)

	vncFileServer := http.FileServer(http.FS(vncFS))
	assetFileServer := http.FileServer(http.FS(assetFS))

	if metricsEnable {
		mdlw = middleware.New(middleware.Config{
			Recorder: metrics.NewRecorder(metrics.Config{}),
		})

		mux.Handle("GET /vms", HTTPLogger(middlewarestd.Handler("/vms", mdlw, NewVMsHandler())))
		mux.Handle("GET /vm/{nameOrID}", HTTPLogger(middlewarestd.Handler("/vm/:nameOrID", mdlw, NewVMHandler())))
		mux.Handle("POST /vm/{nameOrID}/start", HTTPLogger(middlewarestd.Handler("/vm/{nameOrID}/start", mdlw, NewVMStartHandler()))) //nolint:lll
		mux.Handle("POST /vm/{nameOrID}/stop", HTTPLogger(middlewarestd.Handler("/vm/{nameOrID}/stop", mdlw, NewVMStopHandler())))    //nolint:lll
		mux.Handle("GET /vnc/", HTTPLogger(middlewarestd.Handler("/vnc/", mdlw, NoCache(vncFileServer))))
		mux.Handle("GET /assets/", HTTPLogger(middlewarestd.Handler("/assets/", mdlw, assetFileServer)))

		setupMetrics(metricsHost, metricsPort)
	} else {
		mux.Handle("GET /vms", HTTPLogger(NewVMsHandler()))
		mux.Handle("GET /vm/{nameOrID}", HTTPLogger(NewVMHandler()))
		mux.Handle("POST /vm/{nameOrID}/start", HTTPLogger(NewVMStartHandler()))
		mux.Handle("POST /vm/{nameOrID}/stop", HTTPLogger(NewVMStopHandler()))
		mux.Handle("GET /vnc/", HTTPLogger(NoCache(vncFileServer)))
		mux.Handle("GET /assets/", HTTPLogger(assetFileServer))
	}

	go StartGoWebSockifyHTTP()

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         net.JoinHostPort(listenHost, strconv.FormatUint(listenPort, 10)),
		Handler:      mux,
	}

	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
