package rbc

import (
	"errors"
	"sync"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type (
	ValRequest struct {
		RootHash []byte
		Branch   []byte
		Block    [][]byte
	}

	EchoRequest struct {
		ValRequest
	}

	ReadyRequest struct {
		RootHash []byte
	}
)

var ErrNoIdMatchingRequest = errors.New("id is not found.")
var ErrInvalidReqType = errors.New("request is not matching with type.")

// It means it is abstracted as Request interface (cleisthenes/request.go)
func (r ValRequest) Recv()   {}
func (r EchoRequest) Recv()  {}
func (r ReadyRequest) Recv() {}

// Received request
type (
	ValReqRepository struct {
		lock sync.RWMutex
		recv map[cleisthenes.ConnId]*ValRequest
	}

	EchoReqRepository struct {
		lock sync.RWMutex
		recv map[cleisthenes.ConnId]*EchoRequest
	}

	ReadyReqRepository struct {
		lock sync.RWMutex
		recv map[cleisthenes.ConnId]*ReadyRequest
	}
)

type (
	// InnerMessage used in channel when received message
	InnerMessage struct {
		senderId cleisthenes.ConnId
		msg      *pb.Message
		err      error
	}

	// InnerRequest used in channel when send messages
	InnerRequest struct {
		msgs []*pb.Message
		err  error
	}
)

func (r *EchoReqRepository) Save(connId cleisthenes.ConnId, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	echoReq, ok := req.(*EchoRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[connId] = echoReq
	return nil
}
func (r *EchoReqRepository) Find(connId cleisthenes.ConnId) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[connId]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[connId], nil

}
func (r *EchoReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}
func NewEchoReqRepository() (*EchoReqRepository, error) {

	return &EchoReqRepository{
		recv: make(map[cleisthenes.ConnId]*EchoRequest),
		lock: sync.RWMutex{},
	}, nil
}
func (r *ValReqRepository) Save(connId cleisthenes.ConnId, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	valReq, ok := req.(*ValRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[connId] = valReq
	return nil
}
func (r *ValReqRepository) Find(connId cleisthenes.ConnId) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[connId]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[connId], nil

}
func (r *ValReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()
	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}
func NewValReqRepository() (*ValReqRepository, error) {
	return &ValReqRepository{
		recv: make(map[cleisthenes.ConnId]*ValRequest),
		lock: sync.RWMutex{},
	}, nil
}

func (r *ReadyReqRepository) Save(connId cleisthenes.ConnId, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	readyReq, ok := req.(*ReadyRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[connId] = readyReq
	return nil
}
func (r *ReadyReqRepository) Find(connId cleisthenes.ConnId) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[connId]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[connId], nil

}
func (r *ReadyReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}
func NewReadyReqRepository() (*ReadyReqRepository, error) {
	return &ReadyReqRepository{
		recv: make(map[cleisthenes.ConnId]*ReadyRequest),
		lock: sync.RWMutex{},
	}, nil
}
