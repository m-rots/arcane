package main

import (
	"encoding/base64"
	"encoding/binary"
	"io"
	"log/slog"
	"net/http"
)

func rewrite(msg []byte) []byte {
	questions := binary.BigEndian.Uint16(msg[4:])

	end := len(msg)
	answers := uint16(0)
	readOffset := 12

	for i := 0; i < int(questions); i++ {
		pointer := uint16(readOffset) | 0b11<<14

		for {
			len := msg[readOffset]
			readOffset += int(len) + 1

			if len == 0 {
				break
			}
		}

		qtype := binary.BigEndian.Uint16(msg[readOffset:])
		readOffset += 4

		if qtype == 1 {
			answers++

			msg = binary.BigEndian.AppendUint16(msg, pointer)
			msg = append(msg,
				0, 1, // Type (A)
				0, 1, // Class
				0, 0, 14, 16, // TTL
				0, 4, // RD Length
				63, 33, 92, 165, // IPv4
			)
		}
	}

	output := make([]byte, 0, 1024)
	output = append(output, msg[:readOffset]...)    // header + questions
	output = append(output, msg[end:]...)           // answers
	output = append(output, msg[readOffset:end]...) // additional

	// Rewrite Header
	flags := binary.BigEndian.Uint16(output[2:])
	flags &= 0b0111_1001_0000_0000
	flags |= 0b1000_0000_0000_0000

	binary.BigEndian.PutUint16(output[2:], flags)
	binary.BigEndian.PutUint16(output[6:], answers)

	return output
}

func writeResponse(w http.ResponseWriter, msg []byte) {
	slog.Info("rewrote dns")

	w.Header().Set("content-type", "application/dns-message")
	w.Write(rewrite(msg))
}

func dnsHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/dns-query" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch req.Method {
	case "GET":
		query := req.URL.Query()
		base64msg := query.Get("dns")

		msg, err := base64.RawURLEncoding.DecodeString(base64msg)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		writeResponse(w, msg)
	case "POST":
		msg, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		writeResponse(w, msg)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
