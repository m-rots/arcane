package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	err := http.ListenAndServeTLS(":443", "cert.pem", "key.pem", ServerNameHandler{})
	slog.Error("listening failed", "err", err)
}

type ServerNameHandler struct{}

func (h ServerNameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("path", r.URL.Path)

	switch r.TLS.ServerName {
	case "api.ticketswap.com":
		ticketLogger := logger.With(
			"device", r.Header.Get("device-id"),
			"op", r.Header.Get("x-apollo-operation-name"),
		)

		if err := ticketswapHandler(w, r); err != nil {
			ticketLogger.Error("ticketswap", "err", err)
		} else {
			ticketLogger.Info("ticketswap")
		}
	case "dns.arcane.m-rots.com":
		dnsHandler(w, r)

		logger.Info("dns", "method", r.Method)
	default:
		w.WriteHeader(http.StatusNotFound)
		logger.Debug("unknown")
	}
}
