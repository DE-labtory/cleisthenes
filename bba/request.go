package bba

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

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
		return ErrInvalidType
	}
	r.reqMap[addr] = bvalReq
	return nil
}

func (r *bvalReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	req, ok := r.reqMap[addr]
	if !ok {
		return nil, ErrNoResult
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
		return ErrInvalidType
	}
	r.reqMap[addr] = auxReq
	return nil
}

func (r *auxReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	req, ok := r.reqMap[addr]
	if !ok {
		return nil, ErrNoResult
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
	Save(round uint64, addr cleisthenes.Address, req cleisthenes.Request)
	Find(round uint64) map[cleisthenes.Address][]cleisthenes.Request
}

// incomingReqRepsoitory saves incoming messages sent from a node that is already
// in a later epoch. These request will be saved and handled in the next epoch.
type defaultIncomingReqRepository struct {
	lock   sync.RWMutex
	reqMap map[uint64]map[cleisthenes.Address][]cleisthenes.Request
}

func newDefaultIncomingRequestRepository() *defaultIncomingReqRepository {
	return &defaultIncomingReqRepository{
		lock:   sync.RWMutex{},
		reqMap: make(map[uint64]map[cleisthenes.Address][]cleisthenes.Request),
	}
}

func (r *defaultIncomingReqRepository) Save(round uint64, addr cleisthenes.Address, req cleisthenes.Request) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.reqMap[round][addr] == nil {
		r.reqMap[round] = make(map[cleisthenes.Address][]cleisthenes.Request)
		r.reqMap[round][addr] = make([]cleisthenes.Request, 0)
	}
	r.reqMap[round][addr] = append(r.reqMap[round][addr], req)
}

func (r *defaultIncomingReqRepository) Find(round uint64) map[cleisthenes.Address][]cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	result, ok := r.reqMap[round]
	if !ok {
		return map[cleisthenes.Address][]cleisthenes.Request{}
	}
	return result
}
