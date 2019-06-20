package bba

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

type request struct {
	sender cleisthenes.Member
	data   cleisthenes.Request
	round  uint64
	err    chan error
}

type BBA struct {
	*sync.RWMutex
	cleisthenes.Tracer

	owner    cleisthenes.Member
	proposer cleisthenes.Member
	// number of network nodes
	n int
	// number of byzantine nodes which can tolerate
	f int

	stopFlag int32
	// done is flag whether BBA is terminated or not
	done *cleisthenes.BinaryState
	// round is value for current Binary Byzantine Agreement round
	round       uint64
	binValueSet *binarySet
	// broadcastedBvalSet is set of bval value instance has sent
	broadcastedBvalSet *binarySet
	// est is estimated value of BBA instance, dec is decision value
	est, dec       *cleisthenes.BinaryState
	auxBroadcasted *cleisthenes.BinaryState

	bvalRepo        cleisthenes.RequestRepository
	auxRepo         cleisthenes.RequestRepository
	incomingReqRepo incomingRequestRepository

	reqChan             chan request
	closeChan           chan struct{}
	binValueChan        chan struct{}
	tryoutAgreementChan chan struct{}
	advanceRoundChan    chan struct{}

	broadcaster   cleisthenes.Broadcaster
	coinGenerator cleisthenes.CoinGenerator
	binInputChan  cleisthenes.BinarySender
}

func New(
	n int,
	f int,
	owner cleisthenes.Member,
	proposer cleisthenes.Member,
	broadcaster cleisthenes.Broadcaster,
	coinGenerator cleisthenes.CoinGenerator,
	binInputChan cleisthenes.BinarySender,
) *BBA {
	instance := &BBA{
		owner:    owner,
		proposer: proposer,
		n:        n,
		f:        f,
		round:    1,

		binValueSet:        newBinarySet(),
		broadcastedBvalSet: newBinarySet(),

		done: cleisthenes.NewBinaryState(),
		est:  cleisthenes.NewBinaryState(),
		dec:  cleisthenes.NewBinaryState(),

		auxBroadcasted: cleisthenes.NewBinaryState(),

		bvalRepo:        newBvalReqRepository(),
		auxRepo:         newAuxReqRepository(),
		incomingReqRepo: newDefaultIncomingRequestRepository(),

		closeChan:           make(chan struct{}, 1),
		reqChan:             make(chan request, n),
		binValueChan:        make(chan struct{}, n),
		tryoutAgreementChan: make(chan struct{}, n),
		advanceRoundChan:    make(chan struct{}, n),

		broadcaster:   broadcaster,
		coinGenerator: coinGenerator,
		binInputChan:  binInputChan,

		Tracer: cleisthenes.NewMemCacheTracer(),
	}
	go instance.run()
	return instance
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (bba *BBA) HandleInput(bvalRequest *BvalRequest) error {
	bba.est.Set(bvalRequest.Value)

	req := request{
		sender: bba.owner,
		data:   bvalRequest,
		round:  bba.round,
		err:    make(chan error),
	}
	if err := bba.broadcast(bvalRequest); err != nil {
		return err
	}
	bba.reqChan <- req
	return <-req.err
}

// HandleMessage will process the given rpc message.
func (bba *BBA) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	req, round, err := processMessage(msg)
	if err != nil {
		return err
	}
	r := request{
		sender: sender,
		data:   req,
		round:  round,
		err:    make(chan error),
	}
	bba.reqChan <- r
	return <-r.err
}

func (bba *BBA) Result() (cleisthenes.Binary, bool) {
	return bba.dec.Value(), bba.dec.Undefined()
}

func (bba *BBA) Close() {
	if first := atomic.CompareAndSwapInt32(&bba.stopFlag, int32(0), int32(1)); !first {
		return
	}
	bba.closeChan <- struct{}{}
	<-bba.closeChan
	close(bba.reqChan)
}

func (bba *BBA) Trace() {
	bba.Tracer.Trace()
}

func (bba *BBA) muxMessage(sender cleisthenes.Member, round uint64, req cleisthenes.Request) error {
	if round < bba.round {
		return nil
	}
	if round > bba.round {
		bba.incomingReqRepo.Save(round, sender.Address, req)
		return nil
	}

	switch r := req.(type) {
	case *BvalRequest:
		return bba.handleBvalRequest(sender, r)
	case *AuxRequest:
		return bba.handleAuxRequest(sender, r)
	default:
		return ErrUndefinedRequestType
	}
}

func (bba *BBA) handleBvalRequest(sender cleisthenes.Member, bval *BvalRequest) error {
	if err := bba.saveBvalIfNotExist(sender, bval); err != nil {
		return err
	}

	count := bba.countBvalByValue(bval.Value)
	bba.Log("action", "handleBval", "count", strconv.Itoa(count))
	if count == bba.binValueSetThreshold() {
		bba.Log("action", "binValueSet", "count", strconv.Itoa(count))
		bba.binValueSet.union(bval.Value)
		bba.binValueChan <- struct{}{}
		return nil
	}
	if count == bba.bvalBroadcastThreshold() && !bba.broadcastedBvalSet.exist(bval.Value) {
		bba.broadcastedBvalSet.union(bval.Value)
		bba.Log("action", "broadcastBval", "count", strconv.Itoa(count))
		return bba.broadcast(bval)
	}
	return nil
}

func (bba *BBA) handleAuxRequest(sender cleisthenes.Member, aux *AuxRequest) error {
	if err := bba.saveAuxIfNotExist(sender, aux); err != nil {
		return err
	}
	count := bba.countAuxByValue(aux.Value)
	bba.Log("action", "handleAux", "from", sender.Address.String(), "count", strconv.Itoa(count))
	if count < bba.tryoutAgreementThreshold() {
		return nil
	}
	bba.tryoutAgreementChan <- struct{}{}
	return nil
}

func (bba *BBA) tryoutAgreement() {
	bba.Log("action", "tryoutAgreement", "from", bba.owner.Address.String())
	if bba.done.Value() {
		return
	}
	coin := bba.coinGenerator.Coin()

	binList := bba.binValueSet.toList()
	if len(binList) == 0 {
		bba.Log("binary set is empty, but tried agreement")
		return
	}
	if len(binList) > 1 {
		bba.Log("bin value set size is larger than one")
		bba.agreementFailed(cleisthenes.Binary(coin))
		return
	}
	if binList[0] != cleisthenes.Binary(coin) {
		bba.Log("bin value set value is different with coin value")
		bba.agreementFailed(binList[0])
		return
	}
	bba.agreementSuccess(binList[0])
}

func (bba *BBA) agreementFailed(estValue cleisthenes.Binary) {
	bba.est.Set(estValue)
	bba.advanceRoundChan <- struct{}{}
}

func (bba *BBA) agreementSuccess(decValue cleisthenes.Binary) {
	bba.dec.Set(decValue)
	bba.done.Set(true)
	bba.binInputChan.Send(cleisthenes.BinaryMessage{
		Member: bba.owner,
		Binary: bba.dec.Value(),
	})
	bba.advanceRoundChan <- struct{}{}
}

func (bba *BBA) advanceRound() {
	bba.Log("action", "advanceRound", "from", bba.owner.Address.String())
	bba.bvalRepo = newBvalReqRepository()
	bba.auxRepo = newAuxReqRepository()

	bba.binValueSet = newBinarySet()
	bba.broadcastedBvalSet = newBinarySet()

	bba.round++

	bba.handleDelayedRequest(bba.round)

	if err := bba.HandleInput(&BvalRequest{
		Value: bba.est.Value(),
	}); err != nil {
		bba.Log("failed to handle input")
	}
}

func (bba *BBA) handleDelayedRequest(round uint64) {
	delayedReqMap := bba.incomingReqRepo.Find(bba.round)
	for addr, reqList := range delayedReqMap {
		for _, req := range reqList {
			r := request{
				sender: cleisthenes.Member{Address: addr},
				data:   req,
				round:  bba.round,
				err:    make(chan error),
			}
			bba.reqChan <- r
			if err := <-r.err; err != nil {
				bba.Log("action", "handleDelayedRequest", "err", err.Error())
			}
		}
	}
	bba.Log("action", "handleDelayedRequest")
}

func (bba *BBA) broadcastAuxOnceForRound() {
	if bba.auxBroadcasted.Value() == true {
		return
	}
	bba.Log("action", "broadcastAux", "from", bba.owner.Address.String())
	for _, bin := range bba.binValueSet.toList() {
		if err := bba.broadcast(&AuxRequest{
			Value: bin,
		}); err != nil {
			bba.Log("action", "broadcastAux", "err", err.Error())
		}
	}
	bba.auxBroadcasted.Set(true)
}

func (bba *BBA) broadcast(req cleisthenes.Request) error {
	var typ pb.BBAType
	switch req.(type) {
	case *BvalRequest:
		typ = pb.BBA_BVAL
	case *AuxRequest:
		typ = pb.BBA_AUX
	default:
		return errors.New("invalid broadcast message type")
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	bba.broadcaster.ShareMessage(pb.Message{
		Proposer:  bba.proposer.Address.String(),
		Sender:    bba.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Payload: &pb.Message_Bba{
			Bba: &pb.BBA{
				Round:   bba.round,
				Type:    typ,
				Payload: payload,
			},
		},
	})
	return nil
}

func (bba *BBA) toDie() bool {
	return atomic.LoadInt32(&(bba.stopFlag)) == int32(1)
}

func (bba *BBA) run() {
	for !bba.toDie() {
		select {
		case <-bba.closeChan:
			bba.closeChan <- struct{}{}
		case req := <-bba.reqChan:
			req.err <- bba.muxMessage(req.sender, req.round, req.data)
		case <-bba.binValueChan:
			bba.broadcastAuxOnceForRound()
		case <-bba.tryoutAgreementChan:
			bba.tryoutAgreement()
		case <-bba.advanceRoundChan:
			bba.advanceRound()
		}
	}
}

func (bba *BBA) saveBvalIfNotExist(sender cleisthenes.Member, data *BvalRequest) error {
	r, err := bba.bvalRepo.Find(sender.Address)
	if err != nil && !IsErrNoResult(err) {
		return err
	}
	if r != nil {
		return nil
	}
	return bba.bvalRepo.Save(sender.Address, data)
}

func (bba *BBA) saveAuxIfNotExist(sender cleisthenes.Member, data *AuxRequest) error {
	r, err := bba.auxRepo.Find(sender.Address)
	if err != nil && !IsErrNoResult(err) {
		return err
	}
	if r != nil {
		return nil
	}
	return bba.auxRepo.Save(sender.Address, data)
}

func (bba *BBA) countBvalByValue(val cleisthenes.Binary) int {
	bvalList := bba.convToBvalList(bba.bvalRepo.FindAll())
	count := 0
	for _, bval := range bvalList {
		if bval.Value == val {
			count++
		}
	}
	return count
}

func (bba *BBA) countAuxByValue(val cleisthenes.Binary) int {
	auxList := bba.convToAuxList(bba.auxRepo.FindAll())
	count := 0
	for _, aux := range auxList {
		if aux.Value == val {
			count++
		}
	}
	return count
}

func (bba *BBA) convToBvalList(reqList []cleisthenes.Request) []*BvalRequest {
	result := make([]*BvalRequest, 0)
	for _, req := range reqList {
		result = append(result, req.(*BvalRequest))
	}
	return result
}

func (bba *BBA) convToAuxList(reqList []cleisthenes.Request) []*AuxRequest {
	result := make([]*AuxRequest, 0)
	for _, req := range reqList {
		result = append(result, req.(*AuxRequest))
	}
	return result
}

func (bba *BBA) bvalBroadcastThreshold() int {
	return bba.f + 1
}

func (bba *BBA) binValueSetThreshold() int {
	return 2*bba.f + 1
}

func (bba *BBA) tryoutAgreementThreshold() int {
	return bba.n - bba.f
}

func processMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	switch msg.Bba.Type {
	case pb.BBA_BVAL:
		return processBvalMessage(msg)
	case pb.BBA_AUX:
		return processAuxMessage(msg)
	default:
		return nil, 0, errors.New("error processing message with invalid type")
	}
}

func processBvalMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &BvalRequest{}
	if err := json.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func processAuxMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &AuxRequest{}
	if err := json.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}
