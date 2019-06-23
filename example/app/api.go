package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/DE-labtory/cleisthenes"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

type ErrIllegalArgument struct {
	Reason string
}

func (e ErrIllegalArgument) Error() string {
	return fmt.Sprintf("err illegal argument: %s", e.Reason)
}

type endpoint struct {
	logger kitlog.Logger
	hbbft  cleisthenes.Hbbft
}

func newEndpoint(hbbft cleisthenes.Hbbft, logger kitlog.Logger) *endpoint {
	return &endpoint{
		logger: logger,
		hbbft:  hbbft,
	}
}

func NewApiHandler(hbbft cleisthenes.Hbbft, logger kitlog.Logger) http.Handler {
	endpoint := newEndpoint(hbbft, logger)
	r := mux.NewRouter()

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
	}

	r.Methods("GET").Path("/healthz").HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		logger.Log("method", "GET", "endpoint", "healthz")
		w.Write([]byte("up"))
	})

	r.Methods("POST").Path("/tx").Handler(kithttp.NewServer(
		endpoint.proposeTx,
		decodeProposeTxRequest,
		encodeResponse,
		opts...,
	))
	return r
}

func (e *endpoint) proposeTx(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "proposeTx")

	f := e.makeProposeTxEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "proposeTx", "err", err.Error())
	}
	return response, err
}

func (e *endpoint) makeProposeTxEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ProposeTxRequest)
		return ProposeTxResponse{}, e.hbbft.Propose(req.Transaction)
	}
}

type ProposeTxRequest struct {
	Transaction Transaction `json:"transaction"`
}

type ProposeTxResponse struct{}

func decodeProposeTxRequest(_ context.Context, r *http.Request) (interface{}, error) {
	body := ProposeTxRequest{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(body.Transaction, Transaction{}) {
		return nil, ErrIllegalArgument{"transaction is empty"}
	}
	return body, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

type errorer interface {
	error() error
}

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
