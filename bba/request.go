package bba

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type (
	bvalRequest struct {
		Value cleisthenes.Binary
	}

	auxRequest struct {
		Value cleisthenes.Binary
	}
)

func (r bvalRequest) Recv() {}
func (r auxRequest) Recv()  {}

type (
	bvalReqRepository struct {
		reqMap map[cleisthenes.Address]*bvalRequest
	}

	auxReqRepository struct {
		reqMap map[cleisthenes.Address]*auxRequest
	}
)

func newBvalReqRepository() *bvalReqRepository {
	return &bvalReqRepository{
		reqMap: make(map[cleisthenes.Address]*bvalRequest),
	}
}

func (r *bvalReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error { return nil }
func (r *bvalReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	return nil, nil
}
func (r *bvalReqRepository) FindAll() []cleisthenes.Request {
	return nil
}

func newAuxReqRepository() *auxReqRepository {
	return &auxReqRepository{
		reqMap: make(map[cleisthenes.Address]*auxRequest),
	}
}

func (r *auxReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error { return nil }
func (r *auxReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	return nil, nil
}
func (r *auxReqRepository) FindAll() []cleisthenes.Request {
	return nil
}

type incomingRequestRepository interface {
	Save(epoch uint64, addr cleisthenes.Address, req pb.BBA) error
	Find(epoch uint64, addr cleisthenes.Address) pb.BBA
	FindByEpoch(epoch uint64) []pb.BBA
}

// incomingReqRepsoitory saves incoming messages sent from a node that is already
// in a later epoch. These request will be saved and handled in the next epoch.
type defaultIncomingRequestRepository struct {
	reqMap map[uint64]map[cleisthenes.Address]*pb.BBA
}
