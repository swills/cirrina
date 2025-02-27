package main

import (
	"net/http"
	"testing"
)

func Test_newMetricsServer(t *testing.T) {
	mux := http.NewServeMux()

	t.Parallel()

	srv := newMetricsServer("localhost:12345", mux)

	if srv.ReadTimeout == 0 {
		t.Fatal("no metrics server read timeout set")
	}

	if srv.WriteTimeout == 0 {
		t.Fatal("no metrics server write timeout set")
	}
}
