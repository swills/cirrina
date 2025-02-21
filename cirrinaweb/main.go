package main

import (
	"embed"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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
var mdlw *middleware.Middleware

//go:embed assets/*
var assetFS embed.FS

func healthCheck(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusNoContent)
}

// parseEnv parses many different environment variables, not only metrics related things
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

// setupMux sets up the http mux with the middleware for metrics
func setupMux(mux *http.ServeMux, pattern string, handler http.Handler, mdlw *middleware.Middleware) {
	if mdlw != nil {
		h := strings.Split(pattern, " ")
		if len(h) != 2 {
			panic("wrong pattern in mux setup")
		}

		handlerID := strings.ReplaceAll(h[1], "{", ":")
		handlerID = strings.ReplaceAll(handlerID, "}", "")

		mux.Handle(pattern, HTTPLogger(middlewarestd.Handler(handlerID, *mdlw, handler)))

		return
	}

	mux.Handle(pattern, HTTPLogger(handler))
}

//nolint:funlen
func main() {
	var err error

	var metricsHost string

	var metricsPort uint16

	// called for side effects too
	metricsHost, metricsPort = parseEnv()

	vncFileServer := http.FileServerFS(vncFS)
	assetFileServer := http.FileServerFS(assetFS)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthCheck)
	mux.HandleFunc("GET /favicon.ico", handlers.FaviconHandlerFunc)

	// no metrics on these
	mux.Handle("GET /assets/", HTTPLogger(assetFileServer))
	mux.Handle("GET /vnc/", HTTPLogger(NoCache(vncFileServer)))

	if metricsEnable {
		mdlwT := middleware.New(middleware.Config{
			Recorder: metrics.NewRecorder(metrics.Config{}),
		})

		mdlw = &mdlwT
	}

	setupMux(mux, "GET /", handlers.NewHomeHandler(), mdlw)
	setupMux(mux, "GET /home", handlers.NewHomeHandler(), mdlw)

	setupMux(mux, "GET /vms", handlers.NewVMsHandler(), mdlw)

	setupMux(mux, "GET /vm/{nameOrID}", handlers.NewVMHandler(), mdlw)
	setupMux(mux, "DELETE /vm/{nameOrID}", handlers.NewVMHandler(), mdlw)

	setupMux(mux, "POST /vm/{nameOrID}/start", handlers.NewVMStartHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/stop", handlers.NewVMStopHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/clearuefi", handlers.NewVMClearUEFIHandler(), mdlw)

	setupMux(mux, "GET /vm/{nameOrID}/editBasic", handlers.NewVMEditBasicHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/editBasic", handlers.NewVMEditBasicHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editDisk", handlers.NewVMEditDiskHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editISOs", handlers.NewVMEditISOHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editNICs", handlers.NewVMEditNICHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editSerial", handlers.NewVMEditSerialHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/editSerial", handlers.NewVMEditSerialHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editDisplay", handlers.NewVMEditDisplayHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/editDisplay", handlers.NewVMEditDisplayHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editAudio", handlers.NewVMEditAudioHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/editAudio", handlers.NewVMEditAudioHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editStart", handlers.NewVMEditStartHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/editStart", handlers.NewVMEditStartHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/editAdvanced", handlers.NewVMEditAdvancedHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/editAdvanced", handlers.NewVMEditAdvancedHandler(), mdlw)

	setupMux(mux, "GET /vm/{nameOrID}/disk/add", handlers.NewVMDiskAddHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/disk/add", handlers.NewVMDiskAddHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/iso/add", handlers.NewVMISOAddHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/iso/add", handlers.NewVMISOAddHandler(), mdlw)
	setupMux(mux, "GET /vm/{nameOrID}/nic/add", handlers.NewVMNICAddHandler(), mdlw)
	setupMux(mux, "POST /vm/{nameOrID}/nic/add", handlers.NewVMNICAddHandler(), mdlw)

	setupMux(mux, "DELETE /vm/{vmNameOrID}/disk/{diskNameOrID}", handlers.NewVMDiskHandler(), mdlw)
	setupMux(mux, "DELETE /vm/{vmNameOrID}/iso/{isoNameOrID}", handlers.NewVMISOHandler(), mdlw)
	setupMux(mux, "DELETE /vm/{vmNameOrID}/nic/{nicNameOrID}", handlers.NewVMNICHandler(), mdlw)

	setupMux(mux, "GET /vmdata/{nameOrID}", handlers.NewVMDataHandler(), mdlw)

	setupMux(mux, "GET /media/disks", handlers.NewDisksHandler(), mdlw)

	setupMux(mux, "GET /media/disk", handlers.NewDiskHandler(), mdlw)
	setupMux(mux, "POST /media/disk", handlers.NewDiskHandler(), mdlw)
	setupMux(mux, "GET /media/disk/{nameOrID}", handlers.NewDiskHandler(), mdlw)
	setupMux(mux, "DELETE /media/disk/{nameOrID}", handlers.NewDiskHandler(), mdlw)

	setupMux(mux, "GET /media/isos", handlers.NewISOsHandler(), mdlw)
	setupMux(mux, "GET /media/iso/{nameOrID}", handlers.NewISOHandler(), mdlw)
	setupMux(mux, "DELETE /media/iso/{nameOrID}", handlers.NewISOHandler(), mdlw)

	setupMux(mux, "GET /net/nics", handlers.NewNICsHandler(), mdlw)

	setupMux(mux, "GET /net/nic", handlers.NewNICHandler(), mdlw)
	setupMux(mux, "POST /net/nic", handlers.NewNICHandler(), mdlw)
	setupMux(mux, "GET /net/nic/{nameOrID}", handlers.NewNICHandler(), mdlw)
	setupMux(mux, "DELETE /net/nic/{nameOrID}", handlers.NewNICHandler(), mdlw)
	setupMux(mux, "DELETE /net/nic/{nameOrID}/uplink", handlers.NewNICUplinkHandler(), mdlw)
	setupMux(mux, "GET /net/nic/{nameOrID}/uplink", handlers.NewNICUplinkHandler(), mdlw)
	setupMux(mux, "POST /net/nic/{nameOrID}/uplink", handlers.NewNICUplinkHandler(), mdlw)

	setupMux(mux, "GET /net/switches", handlers.NewSwitchesHandler(), mdlw)

	setupMux(mux, "GET /net/switch", handlers.NewSwitchHandler(), mdlw)
	setupMux(mux, "POST /net/switch", handlers.NewSwitchHandler(), mdlw)
	setupMux(mux, "GET /net/switch/{nameOrID}", handlers.NewSwitchHandler(), mdlw)
	setupMux(mux, "DELETE /net/switch/{nameOrID}", handlers.NewSwitchHandler(), mdlw)

	if metricsEnable {
		setupMetrics(metricsHost, metricsPort)
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
