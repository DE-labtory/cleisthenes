package bba

import (
	"errors"
	"sync"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

var ErrNoIdMatchingRequest = errors.New("id is not found.")
var ErrInvalidReqType = errors.New("request is not matching with type.")

type (
	BvalRequest struct {
		Value cleisthenes.Binary
	}

	AuxRequest struct {
		Value cleisthenes.Binary
	}
)

func (r BvalRequest) Recv() {}
func (r AuxRequest) Recv()  {}

type (
	bvalReqRepository struct {
		lock   sync.RWMutex
		reqMap map[cleisthenes.Address]*BvalRequest
	}

	auxReqRepository struct {
		lock   sync.RWMutex
		reqMap map[cleisthenes.Address]*AuxRequest
	}
)

func newBvalReqRepository() *bvalReqRepository {
	return &bvalReqRepository{
		reqMap: make(map[cleisthenes.Address]*BvalRequest),
	}
}

func (r *bvalReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	bvalReq, ok := req.(*BvalRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.reqMap[addr] = bvalReq
	return nil
}
func (r *bvalReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	req, ok := r.reqMap[addr]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return req, nil
}
func (r *bvalReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.reqMap {
		reqList = append(reqList, request)
	}
	return reqList
}

func newAuxReqRepository() *auxReqRepository {
	return &auxReqRepository{
		reqMap: make(map[cleisthenes.Address]*AuxRequest),
	}
}

func (r *auxReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	auxReq, ok := req.(*AuxRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.reqMap[addr] = auxReq
	return nil
}
func (r *auxReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	req, ok := r.reqMap[addr]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return req, nil
}
func (r *auxReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.reqMap {
		reqList = append(reqList, request)
	}
	return reqList
}

type incomingRequestRepository interface {
	Save(epoch uint64, addr cleisthenes.Address, req pb.BBA) error
	Find(epoch uint64, addr cleisthenes.Address) pb.BBA
	FindByEpoch(epoch uint64) []pb.BBA
}

// incomingReqRepsoitory saves incoming messages sent from a node that is already
// in a later epoch. These request will be saved and handled in the next epoch.
type defaultIncomingReqRepository struct {
	reqMap map[uint64]map[cleisthenes.Address]*pb.BBA
}

func newDefaultIncomingRequestRepository() *defaultIncomingReqRepository {
	return &defaultIncomingReqRepository{
		reqMap: make(map[uint64]map[cleisthenes.Address]*pb.BBA),
	}
}

func (r *defaultIncomingReqRepository) Save(epoch uint64, addr cleisthenes.Address, req pb.BBA) error {
	return nil
}
func (r *defaultIncomingReqRepository) Find(epoch uint64, addr cleisthenes.Address) pb.BBA {
	return pb.BBA{}
}
func (r *defaultIncomingReqRepository) FindByEpoch(epoch uint64) []pb.BBA {
	return nil
}
