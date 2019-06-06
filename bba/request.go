package bba

import "github.com/DE-labtory/cleisthenes"

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

// incomingReqRepsoitory saves incoming messages sent from a node that is already
// in a later epoch. These request will be saved and handled in the next epoch.
type incomingReqRepository struct {
	reqMap map[int]map[cleisthenes.Address]cleisthenes.Request
}
