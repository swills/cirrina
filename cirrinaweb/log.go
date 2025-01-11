package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"cirrina/cirrinaweb/util"
)

var _ http.ResponseWriter = &loggingResponseWriter{}

type loggingResponseWriter struct {
	http.ResponseWriter
	HTTPStatus   int
	ResponseSize int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.HTTPStatus = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Flush() {
	z := w.ResponseWriter
	if f, ok := z.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *loggingResponseWriter) Write(bytes []byte) (int, error) {
	if w.HTTPStatus == 0 {
		w.HTTPStatus = 200
	}

	w.ResponseSize = len(bytes)

	n, err := w.ResponseWriter.Write(bytes)
	if err != nil {
		return n, fmt.Errorf("error writing log: %w", err)
	}

	return n, nil
}

func HTTPLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		interceptWriter := loggingResponseWriter{writer, 0, 0}
		host, _, _ := net.SplitHostPort(request.RemoteAddr)

		handler.ServeHTTP(&interceptWriter, request)

		accessLog := util.GetAccessLog()

		_, err := accessLog.WriteString(fmt.Sprintf("%s - - [%s] \"%s %s %s\" %d %d %s\n",
			host,
			time.Now().Format("02/Jan/2006:15:04:05 -0700"),
			request.Method,
			request.URL.Path,
			request.Proto,
			interceptWriter.HTTPStatus,
			interceptWriter.ResponseSize,
			request.UserAgent(),
		))
		if err != nil {
			panic(err)
		}
	})
}
