package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, 32*1024)
	},
}

var transport = http.Transport{
	IdleConnTimeout: 0,
}

var prefetched = sync.Map{}

type prefetchedResponse struct {
	body       []byte
	header     http.Header
	statusCode int
}

func ticketswapHandler(w http.ResponseWriter, req *http.Request) error {
	op := req.Header.Get("x-apollo-operation-name")

	switch op {
	case "AddTicketsToCart":
		return addTicketsToCart(w, req)
	case "GetListing":
		return prefetch(w, req)
	default:
		return proxy(w, req)
	}
}

func copyHeaders(w http.ResponseWriter, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

func rewriteRequest(req *http.Request) {
	req.RequestURI = ""
	req.URL.Scheme = "https"
	req.URL.Host = "api.ticketswap.com"
}

func proxy(w http.ResponseWriter, req *http.Request) error {
	rewriteRequest(req)

	res, err := transport.RoundTrip(req)
	if err != nil {
		return err
	}

	// Copy Headers
	copyHeaders(w, req.Header)

	// Copy StatusCode
	w.WriteHeader(res.StatusCode)

	// Copy body through 32kb buffer
	buf := bufPool.Get().([]byte)

	_, err = io.CopyBuffer(w, res.Body, buf)

	res.Body.Close()
	bufPool.Put(buf)

	slog.Info("proxied request")

	return err
}

func addTicketsToCart(w http.ResponseWriter, req *http.Request) error {
	deviceID := req.Header.Get("device-id")

	res, ok := prefetched.LoadAndDelete(deviceID)
	if !ok {
		return proxy(w, req)
	}

	p := res.(prefetchedResponse)

	copyHeaders(w, p.header)
	w.WriteHeader(p.statusCode)
	_, err := w.Write(p.body)

	slog.Info("loaded prefetch")

	return err
}

func prepareBody(body []byte) ([]byte, error) {
	var cartBody cartRequest

	if err := json.Unmarshal(body, &cartBody); err != nil {
		return []byte{}, err
	}

	return json.Marshal(listingBody{
		Variables: listingVariables{
			ID:   base64.StdEncoding.EncodeToString([]byte(cartBody.Variables.ID)),
			Hash: cartBody.Variables.Hash,
		},
	})
}

func prefetch(w http.ResponseWriter, listingReq *http.Request) error {
	deviceID := listingReq.Header.Get("device-id")

	listingReqBody, err := io.ReadAll(listingReq.Body)
	if err != nil {
		return err
	}

	addToCartReqBody, err := prepareBody(listingReqBody)
	if err != nil {
		return err
	}

	addToCartReq := listingReq.Clone(listingReq.Context())

	rewriteRequest(addToCartReq)
	addToCartReq.Body = io.NopCloser(bytes.NewReader(addToCartReqBody))

	addToCartReq.Header.Set("x-apollo-operation-type", "mutation")
	addToCartReq.Header.Set("x-apollo-operation-name", "AddTicketsToCart")
	addToCartReq.Header.Set("content-length", strconv.Itoa(len(addToCartReqBody)))

	addToCartRes, err := transport.RoundTrip(addToCartReq)
	if err != nil {
		return err
	}

	defer addToCartRes.Body.Close()

	addToCartResBody, err := io.ReadAll(addToCartRes.Body)
	if err != nil {
		return err
	}

	prefetched.Store(deviceID, prefetchedResponse{
		body:       addToCartResBody,
		header:     addToCartRes.Header,
		statusCode: addToCartRes.StatusCode,
	})

	slog.Info("stored prefetch")

	listingReq.Body = io.NopCloser(bytes.NewReader(listingReqBody))
	listingReq.Header.Del("accept-encoding")

	return proxy(w, listingReq)
}

type cartRequest struct {
	OperationName string        `json:"operationName"`
	Query         string        `json:"query"`
	Variables     cartVariables `json:"variables"`
}

type cartVariables struct {
	Tickets int    `json:"amountOfTickets"`
	Hash    string `json:"listingHash"`
	ID      string `json:"listingId"`
}

type listingBody struct {
	Variables listingVariables `json:"variables"`
}

type listingVariables struct {
	Hash string `json:"listingHash"`
	ID   string `json:"listingId"`
}
