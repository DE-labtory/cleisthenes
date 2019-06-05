package bba

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

type Binary = bool

const (
	one  Binary = true
	zero        = false
)

type request struct {
	sender cleisthenes.Member
	data   cleisthenes.Request
	err    chan error
}

type BBA struct {
	*sync.RWMutex

	// number of network nodes
	n int

	// number of byzantine nodes
	f int

	proposer cleisthenes.Member

	stop int32
	// done is flag whether BBA is terminated or not
	done bool

	// epoch is current honeybadger protocol epoch
	epoch int
	// round is value for current epoch
	round int

	// sentBvalSet is set of bval value instance has sent
	sentBvalSet binarySet

	// est is estimated value of BBA instance, dec is decision value
	est, dec Binary

	bvalRepo        cleisthenes.RequestRepository
	auxRepo         cleisthenes.RequestRepository
	incomingReqRepo cleisthenes.IncomingRequestRepository

	closeChan chan struct{}
	reqChan   chan request

	bc cleisthenes.Broadcaster
}

func New(n int) *BBA {
	return &BBA{}
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (b *BBA) HandleInput(val Binary) error {
	return nil
}

// HandleMessage will process the given rpc message.
func (b *BBA) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	return nil
}

func (b *BBA) Result() bool {
	return false
}

func (b *BBA) Close() {
	if first := atomic.CompareAndSwapInt32(&b.stop, int32(0), int32(1)); !first {
		return
	}
	b.closeChan <- struct{}{}
}

func (b *BBA) muxRequest(req request) error {
	switch req.data.(type) {
	case *bvalRequest:
		return b.handleBvalRequest(req)
	case *auxRequest:
		return b.handleAuxRequest(req)
	default:
		return errors.New(fmt.Sprintf("unexpected request type"))
	}
	return nil
}

func (b *BBA) handleBvalRequest(msg request) error {
	return nil
}

func (b *BBA) handleAuxRequest(msg request) error {
	return nil
}

func (b *BBA) toDie() bool {
	return atomic.LoadInt32(&(b.stop)) == int32(1)
}

func (b *BBA) run() {
	for !b.toDie() {
		select {
		case <-b.closeChan:
			b.closeChan <- struct{}{}
			return
		case req := <-b.reqChan:
			req.err <- b.muxRequest(req)
		}
	}
}
