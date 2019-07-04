package bba

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes/test/mock"

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

type round struct {
	lock sync.RWMutex
	val  uint64
}

func newRound() *round {
	return &round{
		lock: sync.RWMutex{},
		val:  1,
	}
}

func (r *round) inc() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.val++
}

func (r *round) value() uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.val
}

type BBA struct {
	sync.RWMutex
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
	round       *round
	binValueSet *binarySet
	// broadcastedBvalSet is set of bval value instance has sent
	broadcastedBvalSet *binarySet
	// est is estimated value of BBA instance, dec is decision value
	est, dec       *cleisthenes.BinaryState
	auxBroadcasted *cleisthenes.BinaryState

	bvalRepo            cleisthenes.RequestRepository
	auxRepo             cleisthenes.RequestRepository
	incomingBvalReqRepo incomingRequestRepository
	incomingAuxReqRepo  incomingRequestRepository

	reqChan             chan request
	closeChan           chan struct{}
	binValueChan        chan uint64
	tryoutAgreementChan chan uint64
	advanceRoundChan    chan uint64

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
	binInputChan cleisthenes.BinarySender,
) *BBA {
	instance := &BBA{
		owner:    owner,
		proposer: proposer,
		n:        n,
		f:        f,
		round:    newRound(),

		binValueSet:        newBinarySet(),
		broadcastedBvalSet: newBinarySet(),

		done: cleisthenes.NewBinaryState(),
		est:  cleisthenes.NewBinaryState(),
		dec:  cleisthenes.NewBinaryState(),

		auxBroadcasted: cleisthenes.NewBinaryState(),

		bvalRepo:            newBvalReqRepository(),
		auxRepo:             newAuxReqRepository(),
		incomingBvalReqRepo: newDefaultIncomingRequestRepository(),
		incomingAuxReqRepo:  newDefaultIncomingRequestRepository(),

		closeChan: make(chan struct{}, 1),

		// request channel size as n*8, because each node can have maximum 4 rounds
		// and in each round each node can broadcast at most twice (broadcast BVAL, AUX)
		// so each node should handle 8 requests per node.
		reqChan: make(chan request, n*8),
		// below channel size as n*4, because each node can have maximum 4 rounds
		binValueChan:        make(chan uint64, n*4),
		tryoutAgreementChan: make(chan uint64, n*4),
		advanceRoundChan:    make(chan uint64, n*4),

		broadcaster:   broadcaster,
		coinGenerator: mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		binInputChan:  binInputChan,

		Tracer: cleisthenes.NewMemCacheTracer(),
	}

	go instance.run()
	return instance
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (bba *BBA) HandleInput(bvalRequest *BvalRequest) error {
	bba.Log("action", "handleInput")
	bba.est.Set(bvalRequest.Value)

	req := request{
		sender: bba.owner,
		data:   bvalRequest,
		round:  bba.round.value(),
		err:    make(chan error),
	}
	if err := bba.broadcast(bvalRequest); err != nil {
		return err
	}
	bba.reqChan <- req

	return nil
}

// HandleMessage will process the given rpc message.
func (bba *BBA) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	req, round, err := bba.processMessage(msg)
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
	return nil
}

func (bba *BBA) Result() (cleisthenes.Binary, bool) {
	return bba.dec.Value(), bba.dec.Undefined()
}

func (bba *BBA) Idle() bool {
	return bba.round.value() == 1 && bba.est.Undefined()
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
	if round < bba.round.value() {
		bba.Log(
			"action", "muxMessage",
			"message", "request from old round, abandon",
			"sender", sender.Address.String(),
			"type", reflect.TypeOf(req).String(),
			"req.round", strconv.FormatUint(round, 10),
			"my.round", strconv.FormatUint(bba.round.value(), 10),
		)
		return nil
	}
	if round > bba.round.value() {
		bba.Log(
			"action", "muxMessage",
			"message", "request from future round, keep it",
			"sender", sender.Address.String(),
			"type", reflect.TypeOf(req).String(),
			"req.round", strconv.FormatUint(round, 10),
			"my.round", strconv.FormatUint(bba.round.value(), 10),
		)
		bba.saveIncomingRequest(round, sender, req)
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
	bba.Log("action", "handleBval", "from", sender.Address.String(), "count", strconv.Itoa(count))
	if count == bba.binValueSetThreshold() {
		bba.Log("action", "binValueSet", "count", strconv.Itoa(count))
		bba.binValueSet.union(bval.Value)
		bba.binValueChan <- bba.round.value()
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
	bba.tryoutAgreementChan <- bba.round.value()

	return nil
}

func (bba *BBA) tryoutAgreement() {
	bba.Log("action", "tryoutAgreement", "from", bba.owner.Address.String())
	if bba.done.Value() {
		return
	}

	binList := bba.binValueSet.toList()
	if len(binList) == 0 {
		bba.Log("binary set is empty, but tried agreement")
		return
	}

	coin := bba.coinGenerator.Coin(bba.round.value())

	if len(binList) > 1 {
		bba.Log("err", "bin value set size is larger than one")
		bba.agreementFailed(cleisthenes.Binary(coin))
		return
	}
	if binList[0] != cleisthenes.Binary(coin) {
		bba.Log("err", "bin value set value is different with coin value")
		bba.agreementFailed(binList[0])
		return
	}
	bba.agreementSuccess(binList[0])
}

func (bba *BBA) agreementFailed(estValue cleisthenes.Binary) {
	bba.est.Set(estValue)
	bba.advanceRoundChan <- bba.round.value()
}

func (bba *BBA) agreementSuccess(decValue cleisthenes.Binary) {
	bba.dec.Set(decValue)
	bba.done.Set(true)
	bba.binInputChan.Send(cleisthenes.BinaryMessage{
		Member: bba.proposer,
		Binary: bba.dec.Value(),
	})
	bba.Log(
		"action", "agreementSucess",
		"message", "<agreement finish>",
		"my.round", strconv.FormatUint(bba.round.value(), 10),
	)
	bba.advanceRoundChan <- bba.round.value()
}

func (bba *BBA) advanceRound() {
	bba.Log("action", "advanceRound", "from", bba.owner.Address.String())
	bba.bvalRepo = newBvalReqRepository()
	bba.auxRepo = newAuxReqRepository()

	bba.binValueSet = newBinarySet()
	bba.broadcastedBvalSet = newBinarySet()
	bba.auxBroadcasted = cleisthenes.NewBinaryState()

	bba.round.inc()

	bba.handleDelayedBvalRequest()

	bba.HandleInput(&BvalRequest{
		Value: bba.est.Value(),
	})
}

func (bba *BBA) handleDelayedBvalRequest() {
	delayedReqMap := bba.incomingBvalReqRepo.Find(bba.round.value())
	for _, ir := range delayedReqMap {
		if ir.round != bba.round.value() {
			continue
		}
		bba.Log(
			"action", "handleDelayedRequest",
			"round", strconv.FormatUint(ir.round, 10),
			"size", strconv.Itoa(len(delayedReqMap)),
			"addr", ir.addr.String(),
			"type", reflect.TypeOf(ir.req).String(),
		)
		r := request{
			sender: cleisthenes.Member{Address: ir.addr},
			data:   ir.req,
			round:  ir.round,
			err:    make(chan error),
		}
		bba.reqChan <- r
	}
	bba.Log("action", "handleDelayedRequest", "message", "done")
}

func (bba *BBA) handleDelayedAuxRequest() {
	delayedReqMap := bba.incomingAuxReqRepo.Find(bba.round.value())
	for _, ir := range delayedReqMap {
		if ir.round != bba.round.value() {
			continue
		}
		bba.Log(
			"action", "handleDelayedRequest",
			"round", strconv.FormatUint(ir.round, 10),
			"size", strconv.Itoa(len(delayedReqMap)),
			"addr", ir.addr.String(),
			"type", reflect.TypeOf(ir.req).String(),
		)
		r := request{
			sender: cleisthenes.Member{Address: ir.addr},
			data:   ir.req,
			round:  ir.round,
			err:    make(chan error),
		}
		bba.reqChan <- r
	}
	bba.Log("action", "handleDelayedRequest", "message", "done")
}

func (bba *BBA) broadcastAuxOnceForRound() {
	if bba.auxBroadcasted.Value() == true {
		return
	}
	bba.Log("action", "broadcastAux", "from", bba.owner.Address.String(), "round", strconv.FormatUint(bba.round.value(), 10))
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

	bbaMsg := &pb.Message_Bba{
		Bba: &pb.BBA{
			Round:   bba.round.value(),
			Type:    typ,
			Payload: payload,
		},
	}
	broadcastMsg := pb.Message{
		Proposer:  bba.proposer.Address.String(),
		Sender:    bba.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Payload:   bbaMsg,
	}

	bba.HandleMessage(bba.owner, bbaMsg)
	bba.broadcaster.ShareMessage(broadcastMsg)
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
			bba.muxMessage(req.sender, req.round, req.data)
		case r := <-bba.binValueChan:
			if bba.matchWithCurrentRound(r) {
				bba.broadcastAuxOnceForRound()
				bba.handleDelayedAuxRequest()
			}
		case r := <-bba.tryoutAgreementChan:
			if bba.matchWithCurrentRound(r) {
				bba.tryoutAgreement()
			}
		case r := <-bba.advanceRoundChan:
			if bba.matchWithCurrentRound(r) {
				bba.advanceRound()
			}
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

func (bba *BBA) processMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	switch msg.Bba.Type {
	case pb.BBA_BVAL:
		return bba.processBvalMessage(msg)
	case pb.BBA_AUX:
		return bba.processAuxMessage(msg)
	default:
		return nil, 0, errors.New("error processing message with invalid type")
	}
}

func (bba *BBA) processBvalMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &BvalRequest{}
	if err := json.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) processAuxMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &AuxRequest{}
	if err := json.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) saveIncomingRequest(round uint64, sender cleisthenes.Member, req cleisthenes.Request) {
	switch r := req.(type) {
	case *BvalRequest:
		bba.incomingBvalReqRepo.Save(round, sender.Address, r)
	case *AuxRequest:
		bba.incomingAuxReqRepo.Save(round, sender.Address, r)
	}
}

func (bba *BBA) matchWithCurrentRound(round uint64) bool {
	if round != bba.round.value() {
		return false
	}
	return true
}
