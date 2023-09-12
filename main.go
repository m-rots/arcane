package main

import (
	"log"
	"log/slog"
	"net/http"
)

func main() {
	err := http.ListenAndServeTLS(":443", "cert.pem", "key.pem", ServerNameHandler{})
	log.Fatal(err)
}

type ServerNameHandler struct{}

func (h ServerNameHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	slog.Info("request!", "method", req.Method, "server", req.TLS.ServerName, "url", req.URL)

	switch req.TLS.ServerName {
	case "api.ticketswap.com":
		if err := ticketswapHandler(w, req); err != nil {
			slog.Error("ticketswap", "err", err)
		}
	case "dns.arcane.m-rots.com":
		dnsHandler(w, req)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
