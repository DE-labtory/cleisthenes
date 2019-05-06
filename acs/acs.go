package acs

import (
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/rbc"
)

type request struct {
	sender cleisthenes.Member
	data   *pb.Message
	err    chan error
}

type ACS struct {
	owner cleisthenes.Member
	// TODO : add Tracer

	// rbcMap has rbc instances
	// TODO: abstract map into repository
	rbcMap map[cleisthenes.Address]*rbc.RBC

	// bbaMap has bba instances
	// TODO: abstract map into repository
	bbaMap map[cleisthenes.Address]*bba.BBA

	// broadcastResult collects RBC instances' result
	// TODO: abstract map into struct && add lock
	broadcastResult map[cleisthenes.Address][]byte

	// agreementResult collects BBA instances' result
	// each entry have three states: undefined, zero, one
	// TODO: change datastructure into map[cleisthenes.Address]cleisthenes.BinaryState
	// TODO: abstract map into struct && add lock
	agreementResult map[cleisthenes.Address]bool

	// agreementStarted saves whether BBA instances started
	// binary byzantine agreement
	// TODO: change datastructure into map[cleisthenes.Address]cleisthenes.Binary
	// TODO: abstract map into struct && add lock
	agreementStarted map[cleisthenes.Address]bool

	reqChan       chan request
	agreementChan chan struct{}
	closeChan     chan struct{}

	stopFlag int32

	// done sends signal when ACS done its task
	done chan struct{}
}

func New(owner cleisthenes.Member) *ACS {
	return &ACS{
		owner:  owner,
		rbcMap: make(map[cleisthenes.Address]*rbc.RBC),
		bbaMap: make(map[cleisthenes.Address]*bba.BBA),
		// TODO : consider size of reqChan, otherwise this might cause requests to be lost
		reqChan:   make(chan request),
		closeChan: make(chan struct{}, 1),
	}
}

// HandleInput receive batch from honeybadger
func (acs *ACS) HandleInput(batch cleisthenes.Batch) error {
	return nil
}

func (acs *ACS) HandleMessage(sender cleisthenes.Member, msg *pb.Message) error {
	req := request{
		sender: sender,
		data:   msg,
		err:    make(chan error),
	}

	acs.reqChan <- req
	return <-req.err
}

// Result return consensused batch to honeybadger component
func (acs *ACS) Result() cleisthenes.Batch {
	return cleisthenes.Batch{}
}

func (acs *ACS) Close() {
	if first := atomic.CompareAndSwapInt32(&acs.stopFlag, int32(0), int32(1)); !first {
		return
	}
	acs.closeChan <- struct{}{}
	<-acs.closeChan
	close(acs.closeChan)
}

func (acs *ACS) muxMessage(sender cleisthenes.Member, msg *pb.Message) error {
	switch pl := msg.Payload.(type) {
	case *pb.Message_Rbc:
		return acs.handleRbcMessage(sender, pl)
	case *pb.Message_Bba:
		return acs.handleBbaMessage(sender, pl)
	default:
		return cleisthenes.ErrUndefinedRequestType
	}
}

func (acs *ACS) handleRbcMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	// TODO: provide input to RBC
	// TODO: if BA not provided value, then provide 1
	// TODO: if N-f BA received 1, then signal to agreementChan
	return nil
}

func (acs *ACS) handleBbaMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	// TODO: provide input to BBA instance
	return nil
}

// inputZeroToIdleBba send zero to bba instances which still do
// not receive input value
func (acs *ACS) sendZeroToIdleBba() {}

func (acs *ACS) run() {
	for !acs.toDie() {
		select {
		case <-acs.closeChan:
			acs.closeChan <- struct{}{}
		case req := <-acs.reqChan:
			req.err <- acs.muxMessage(req.sender, req.data)
		case <-acs.agreementChan:
			acs.sendZeroToIdleBba()
		}
	}
}

func (acs *ACS) toDie() bool {
	return atomic.LoadInt32(&(acs.stopFlag)) == int32(1)
}
