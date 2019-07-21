package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/DE-labtory/cleisthenes/core"

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
	hbbft  core.Hbbft
}

func newEndpoint(hbbft core.Hbbft, logger kitlog.Logger) *endpoint {
	return &endpoint{
		logger: logger,
		hbbft:  hbbft,
	}
}

func NewApiHandler(hbbft core.Hbbft, logger kitlog.Logger) http.Handler {
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

	r.Methods("POST").Path("/connections").Handler(kithttp.NewServer(
		endpoint.createConnections,
		decodeCreateConnectionsRequest,
		encodeResponse,
		opts...,
	))

	r.Methods("GET").Path("/connections").Handler(kithttp.NewServer(
		endpoint.getConnections,
		decodeGetConnectionsRequest,
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
		return ProposeTxResponse{}, e.hbbft.Submit(req.Transaction)
	}
}

func (e *endpoint) createConnections(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "createConnections")

	f := e.makeCreateConnectionsEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "createConnections", "err", err.Error())
	}
	return response, err
}

func (e *endpoint) makeCreateConnectionsEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateConnectionsRequest)
		return CreateConnectionsResponse{}, e.hbbft.ConnectAll(req.TargetList)
	}
}

func (e *endpoint) getConnections(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "getConnections")

	f := e.makeGetConnectionsEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "getConnections", "err", err.Error())
	}
	return response, err
}

func (e *endpoint) makeGetConnectionsEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return GetConnectionsResponse{
			ConnectionList: e.hbbft.ConnectionList(),
		}, nil
	}
}

type ProposeTxRequest struct {
	Transaction Transaction `json:"transaction"`
}

type ProposeTxResponse struct{}

type CreateConnectionsRequest struct {
	TargetList []string `json:"targets"`
}

type CreateConnectionsResponse struct{}

type GetConnectionsRequest struct{}

type GetConnectionsResponse struct {
	ConnectionList []string `json:"connections"`
}

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

func decodeCreateConnectionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	body := CreateConnectionsRequest{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func decodeGetConnectionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return GetConnectionsRequest{}, nil
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
