package handlers

import (
	_ "embed"
	"net/http"
)

//go:embed favicon.ico
var favicon []byte

func FaviconHandlerFunc(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(favicon)
}
