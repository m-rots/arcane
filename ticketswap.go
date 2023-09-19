package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strconv"
)

const apiURL string = "https://api.ticketswap.com/graphql/public"

//go:embed addTicketsToCart.gql
var addTicketsToCartQuery string

var reverseProxy = httputil.ReverseProxy{
	Director: func(r *http.Request) {
		r.Header["X-Forwarded-For"] = nil
		r.URL.Host = "api.ticketswap.com"
		r.URL.Scheme = "https"
	},
}

func ticketswapHandler(w http.ResponseWriter, req *http.Request) error {
	op := req.Header.Get("x-apollo-operation-name")

	switch op {
	case "GetListing":
		return prefetch(w, req)
	default:
		reverseProxy.ServeHTTP(w, req)
		return nil
	}
}

func base64ListingID(id string) string {
	// If number, then base64 encode it
	if _, err := strconv.Atoi(id); err == nil {
		prefixed := fmt.Sprintf("Listing:%s")

		return base64.StdEncoding.EncodeToString([]byte(prefixed))
	}

	return id
}

func prepareBody(body []byte) ([]byte, error) {
	var listingBody listingRequest

	if err := json.Unmarshal(body, &listingBody); err != nil {
		return []byte{}, err
	}

	return json.Marshal(cartRequest{
		OperationName: "AddTicketsToCart",
		Query:         addTicketsToCartQuery,
		Variables: cartVariables{
			ID:      base64ListingID(listingBody.Variables.ID),
			Hash:    listingBody.Variables.Hash,
			Tickets: 1,
		},
	})
}

func prefetch(w http.ResponseWriter, listingReq *http.Request) error {
	ctx := listingReq.Context()

	listingReqBody, err := io.ReadAll(listingReq.Body)
	if err != nil {
		return err
	}

	addToCartReqBody, err := prepareBody(listingReqBody)
	if err != nil {
		return err
	}

	addToCartReader := bytes.NewReader(addToCartReqBody)
	addToCartReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, addToCartReader)
	if err != nil {
		return err
	}

	addToCartReq.Header = listingReq.Header.Clone()
	addToCartReq.Header.Set("x-apollo-operation-type", "mutation")
	addToCartReq.Header.Set("x-apollo-operation-name", "AddTicketsToCart")

	addToCartRes, err := http.DefaultTransport.RoundTrip(addToCartReq)
	if err != nil {
		return err
	}

	addToCartRes.Body.Close()
	slog.Info("put a thing in a cart! maybe?")

	listingReq.Body = io.NopCloser(bytes.NewReader(listingReqBody))
	reverseProxy.ServeHTTP(w, listingReq)
	return nil
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

type listingRequest struct {
	Variables listingVariables `json:"variables"`
}

type listingVariables struct {
	Hash string `json:"listingHash"`
	ID   string `json:"listingId"`
}
