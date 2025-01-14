package main

import (
	"embed"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"

	"cirrina/cirrinaweb/handlers"
	"cirrina/cirrinaweb/util"
)

//go:embed vnc/*
var vncFS embed.FS

var (
	goWebSockifyBytesTx = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "go_websockify",
			Name:      "websocket_bytes_tx_total",
			Help:      "websocket connection bytes transmitted",
		},
	)

	goWebSockifyBytesRx = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "go_websockify",
			Name:      "websocket_bytes_rx_total",
			Help:      "websocket connection bytes received",
		},
	)

	goWebSockifyWSConnCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "go_websockify",
			Name:      "websocket_connections_active",
			Help:      "Active WebSocket connections",
		})

	goWebSockifyTCPConnCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "go_websockify",
			Name:      "tcp_connections_active",
			Help:      "Active TCP connections",
		})
)

func init() {
	prometheus.MustRegister(goWebSockifyBytesTx)
	prometheus.MustRegister(goWebSockifyBytesRx)
	prometheus.MustRegister(goWebSockifyWSConnCounter)
	prometheus.MustRegister(goWebSockifyTCPConnCounter)
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  65536, // FIXME - config
		WriteBufferSize: 65536, // FIXME - config
		CheckOrigin:     authenticateOrigin,
		Subprotocols:    []string{"binary"},
	}
	server = &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
)

// StartGoWebSockifyHTTP starts the Go WebSockify web server.
func StartGoWebSockifyHTTP() {
	router := http.NewServeMux()
	router.HandleFunc("/", webSocketHandler)

	server = &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Addr:              net.JoinHostPort(util.GetListenHost(), strconv.Itoa(cast.ToInt(util.GetWebsockifyPort()))),
		Handler:           router,
	}

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// authenticateOrigin parses an HTTP request and checks
// if it's a valid request according to rules
func authenticateOrigin(_ *http.Request) bool {
	// TODO
	return true
}

// webSocketHandler handles an incoming HTTP upgrade request
// and starts a bidirectional proxy to the remote connection.
func webSocketHandler(writer http.ResponseWriter, request *http.Request) {
	wsConn, err := upgrader.Upgrade(writer, request, nil)

	if err != nil {
		return
	}

	vmNameOrID := strings.TrimLeft(request.URL.Path, "/")

	aVM, err := handlers.GetVM(vmNameOrID)
	if err != nil {
		return
	}

	if aVM.VNCPort == 0 {
		return
	}

	host, port, err := net.SplitHostPort(net.JoinHostPort(util.GetServerName(), strconv.FormatUint(aVM.VNCPort, 10)))
	if err != nil {
		return
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		message := "failed to resolve destination: " + err.Error()
		_ = wsConn.WriteMessage(websocket.CloseMessage, []byte(message))

		return
	}

	var proxy Proxy = new(ProxyServer)

	proxy.Initialize(wsConn, tcpAddr)

	if err := proxy.Dial(); err != nil {
		return
	}

	go proxy.Start()
}

// Proxy interface
type Proxy interface {
	Initialize(wsConn *websocket.Conn, tcpAddr *net.TCPAddr) *ProxyServer
	Start()
	Dial() error
}

// ProxyServer holds state information about the connection
// being proxied.
type ProxyServer struct {
	wsConn       *websocket.Conn
	tcpAddr      *net.TCPAddr
	tcpConn      *net.TCPConn
	mu           sync.Mutex
	wsConnected  bool
	tcpConnected bool
}

// Initialize ProxyServer and return struct.
func (p *ProxyServer) Initialize(wsConn *websocket.Conn, tcpAddr *net.TCPAddr) *ProxyServer {
	p.wsConn = wsConn
	p.tcpAddr = tcpAddr
	p.mu.Lock()
	p.wsConnected = true
	p.mu.Unlock()
	goWebSockifyWSConnCounter.Inc()

	return p
}

// Start the bidirectional communication channel
// between the WebSocket and the remote connection.
func (p *ProxyServer) Start() {
	go p.ReadWebSocket()
	go p.ReadTCP()
}

// Dial is a function of proxyserver struct that
// instantiates a TCP connection to proxyserver.tcpAddr
func (p *ProxyServer) Dial() error {
	tcpConn, err := net.DialTCP(p.tcpAddr.Network(), nil, p.tcpAddr)

	if err != nil {
		message := "dialing fail: " + err.Error()

		_ = p.wsConn.WriteMessage(websocket.TextMessage, []byte(message))

		return fmt.Errorf("error dialing: %w", err)
	}

	p.tcpConn = tcpConn

	p.mu.Lock()
	p.tcpConnected = true
	p.mu.Unlock()
	goWebSockifyTCPConnCounter.Inc()

	return nil
}

// ReadWebSocket reads from the WebSocket and
// writes to the backend TCP connection.
func (p *ProxyServer) ReadWebSocket() {
	for {
		_, data, err := p.wsConn.ReadMessage()
		if err != nil {
			if p.wsConnected {
				p.mu.Lock()
				p.wsConnected = false
				p.mu.Unlock()
				_ = p.wsConn.Close()

				goWebSockifyWSConnCounter.Dec()
			}

			if p.tcpConnected {
				p.mu.Lock()
				p.tcpConnected = false
				p.mu.Unlock()
				_ = p.tcpConn.Close()

				goWebSockifyTCPConnCounter.Dec()
			}

			break
		}

		_, err = p.tcpConn.Write(data)
		if err != nil {
			_ = p.Dial()
			_, _ = p.tcpConn.Write(data)
		}

		goWebSockifyBytesTx.Add(float64(len(data)))
	}
}

// ReadTCP reads from the backend TCP connection and writes to the WebSocket.
func (p *ProxyServer) ReadTCP() {
	buffer := make([]byte, 65536) // FIXME - config)

	for {
		bytesRead, err := p.tcpConn.Read(buffer)

		if err != nil {
			if p.wsConnected {
				p.mu.Lock()
				p.wsConnected = false
				p.mu.Unlock()
				_ = p.wsConn.Close()

				goWebSockifyWSConnCounter.Dec()
			}

			if p.tcpConnected {
				p.mu.Lock()
				p.tcpConnected = false
				p.mu.Unlock()
				_ = p.tcpConn.Close()

				goWebSockifyTCPConnCounter.Dec()
			}

			break
		}

		if err := p.wsConn.WriteMessage(websocket.BinaryMessage, buffer[:bytesRead]); err != nil {
			break
		}

		goWebSockifyBytesRx.Add(float64(bytesRead))
	}
}

var epoch = time.Unix(0, 0).Format(time.RFC1123)

var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

func NoCache(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// Delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if request.Header.Get(v) != "" {
				request.Header.Del(v)
			}
		}

		// Set our NoCache headers
		for k, v := range noCacheHeaders {
			writer.Header().Set(k, v)
		}

		handler.ServeHTTP(writer, request)
	})
}
