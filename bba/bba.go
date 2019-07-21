package bba

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/iLogger"

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
		val:  0,
	}
}

func (r *round) inc() uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.val++
	return r.val
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

	epoch cleisthenes.Epoch

	stopFlag int32
	// done is flag whether BBA is terminated or not
	done *cleisthenes.BinaryState
	// round is value for current Binary Byzantine Agreement round
	round       *round
	binValueSet []*binarySet
	// broadcastedBvalSet is set of bval value instance has sent
	broadcastedBvalSet []*binarySet
	// est is estimated value of BBA instance, dec is decision value
	est            []*cleisthenes.BinaryState
	dec            *cleisthenes.BinaryState
	auxBroadcasted []*cleisthenes.BinaryState

	bvalRepo            []cleisthenes.RequestRepository
	auxRepo             []cleisthenes.RequestRepository
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
	n, f int,
	epoch cleisthenes.Epoch,
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
		epoch:    epoch,
		round:    newRound(),

		binValueSet:        make([]*binarySet, n+1),
		broadcastedBvalSet: make([]*binarySet, n+1),

		done: cleisthenes.NewBinaryState(),
		est:  make([]*cleisthenes.BinaryState, n+1),
		dec:  cleisthenes.NewBinaryState(),

		bvalRepo: make([]cleisthenes.RequestRepository, n+1),
		auxRepo:  make([]cleisthenes.RequestRepository, n+1),

		auxBroadcasted:      make([]*cleisthenes.BinaryState, n+1),
		incomingBvalReqRepo: newDefaultIncomingRequestRepository(),
		incomingAuxReqRepo:  newDefaultIncomingRequestRepository(),

		closeChan: make(chan struct{}),

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
		Tracer:        cleisthenes.NewMemCacheTracer(),
	}

	for idx := 0; idx <= n; idx++ {
		instance.binValueSet[idx] = newBinarySet()
		instance.broadcastedBvalSet[idx] = newBinarySet()
		instance.est[idx] = cleisthenes.NewBinaryState()
		instance.bvalRepo[idx] = newBvalReqRepository()
		instance.auxRepo[idx] = newAuxReqRepository()
		instance.auxBroadcasted[idx] = cleisthenes.NewBinaryState()
	}

	go instance.run()
	return instance
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (bba *BBA) HandleInput(round uint64, bvalRequest *BvalRequest) error {
	bba.Log("action", "handleInput")
	bba.est[round].Set(bvalRequest.Value)

	req := request{
		sender: bba.owner,
		data:   bvalRequest,
		round:  round,
		err:    make(chan error),
	}

	bba.broadcastedBvalSet[round].union(bvalRequest.Value)
	if err := bba.broadcast(round, bvalRequest); err != nil {
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
	return bba.round.value() == 0 && bba.est[0].Undefined()
}

func (bba *BBA) Round() uint64 {
	return bba.round.value()
}

func (bba *BBA) Close() {
	bba.closeChan <- struct{}{}
	<-bba.closeChan
	if first := atomic.CompareAndSwapInt32(&bba.stopFlag, int32(0), int32(1)); !first {
		return
	}
	//close(bba.closeChan)
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
		return bba.handleBvalRequest(sender, r, round)
	case *AuxRequest:
		return bba.handleAuxRequest(sender, r, round)
	default:
		return ErrUndefinedRequestType
	}
}

func (bba *BBA) handleBvalRequest(sender cleisthenes.Member, bval *BvalRequest, round uint64) error {
	if err := bba.saveBvalIfNotExist(round, sender, bval); err != nil {
		return err
	}

	count := bba.countBvalByValue(round, bval.Value)
	if count == bba.binValueSetThreshold() {
		bba.Log("action", "binValueSet", "count", strconv.Itoa(count))
		bba.binValueSet[round].union(bval.Value)
		bba.binValueChan <- round
		return nil
	}
	if count == bba.bvalBroadcastThreshold() && !bba.broadcastedBvalSet[round].exist(bval.Value) {
		bba.broadcastedBvalSet[round].union(bval.Value)
		bba.Log("action", "broadcastBval", "count", strconv.Itoa(count))
		return bba.broadcast(round, bval)
	}
	return nil
}

func (bba *BBA) handleAuxRequest(sender cleisthenes.Member, aux *AuxRequest, round uint64) error {
	if len(bba.binValueSet[round].toList()) == 0 {
		bba.incomingAuxReqRepo.Save(round, sender.Address, aux)
		return nil
	}
	if err := bba.saveAuxIfNotExist(round, sender, aux); err != nil {
		return err
	}
	count := bba.countAuxByValue(round, aux.Value)

	bba.Log("action", "handleAux", "from", sender.Address.String(), "count", strconv.Itoa(count))
	if count < bba.tryoutAgreementThreshold() {
		return nil
	}

	// TODO : here : 3개에서 4개 올라 갈 일은 없으니까 agreement는 한 round에 한번만하게!
	if count == bba.tryoutAgreementThreshold() {
		bba.tryoutAgreementChan <- round
	}

	return nil
}

func (bba *BBA) tryoutAgreement(round uint64) {
	if !bba.matchWithCurrentRound(round) {
		return
	}
	bba.Log("action", "tryoutAgreement", "from", bba.owner.Address.String())

	if round > 3 {
		return
	}

	binList := bba.binValueSet[round].toList()

	coin := bba.coinGenerator.Coin(round)
	if len(binList) == 0 {
		fmt.Printf("ddddd")
		bba.Log("binary set is empty, but tried agreement")
		return
	}

	if len(binList) > 1 {
		bba.Log("err", "bin value set size is larger than one")
		bba.agreementFailed(round, cleisthenes.Binary(coin))
		return
	}
	if binList[0] != cleisthenes.Binary(coin) {
		bba.Log("err", "bin value set value is different with coin value")
		bba.agreementFailed(round, binList[0])
		return
	}
	bba.agreementSuccess(round, binList[0])
}

func (bba *BBA) agreementFailed(round uint64, estValue cleisthenes.Binary) {
	if round > 3 {
		return
	}
	bba.est[round+1].Set(estValue)
	bba.advanceRoundChan <- round
}

func (bba *BBA) agreementSuccess(round uint64, decValue cleisthenes.Binary) {
	if round > 3 {
		return
	}
	bba.est[round+1].Set(decValue)
	bba.dec.Set(decValue)
	if !bba.done.Value() {
		bba.binInputChan.Send(cleisthenes.BinaryMessage{
			Member: bba.proposer,
			Binary: bba.dec.Value(),
		})
	}
	bba.done.Set(true)
	bba.Log(
		"action", "agreementSucess",
		"message", "<agreement finish>",
		"my.round", strconv.FormatUint(bba.round.value(), 10),
	)

	iLogger.Debugf(nil, "[BBA success] epoch : %d, owner : %s, proposer : %s\n", bba.epoch, bba.owner.Address.String(), bba.proposer.Address.String())
	bba.advanceRoundChan <- round
}

func (bba *BBA) advanceRound(round uint64) {
	if round > 2 {
		return
	}
	bba.Log("action", "advanceRound", "from", bba.owner.Address.String())
	bba.round.inc()
	iLogger.Debugf(nil, "[BBA advance] epoch : %d, round : %d, owner : %s, proposer : %s\n", bba.epoch, round, bba.owner.Address.String(), bba.proposer.Address.String())

	bba.handleDelayedBvalRequest(round + 1)

	bba.HandleInput(round+1, &BvalRequest{
		Value: bba.est[round+1].Value(),
	})
}

func (bba *BBA) handleDelayedBvalRequest(round uint64) {
	delayedReqMap := bba.incomingBvalReqRepo.Find(round)
	for _, ir := range delayedReqMap {
		if ir.round != round {
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

func (bba *BBA) handleDelayedAuxRequest(round uint64) {
	delayedReqMap := bba.incomingAuxReqRepo.Find(round)
	for _, ir := range delayedReqMap {
		if ir.round != round {
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

func (bba *BBA) broadcastAuxOnceForRound(round uint64) {
	bba.Log("action", "broadcastAux", "from", bba.owner.Address.String(), "round", strconv.FormatUint(bba.round.value(), 10))

	if len(bba.binValueSet[round].toList()) == 2 {
		if err := bba.broadcast(round,
			&AuxRequest{
				Value: cleisthenes.One,
			}); err != nil {
			bba.Log("action", "broadcastAux", "binValue length", strconv.Itoa(2), "err", err.Error())
		}
		return
	}

	if len(bba.binValueSet[round].toList()) == 1 {
		bin := bba.binValueSet[round].toList()[0]
		if err := bba.broadcast(round,
			&AuxRequest{
				Value: bin,
			}); err != nil {
			bba.Log("action", "broadcastAux", "binValue length", strconv.Itoa(1), "err", err.Error())
		}
	}
}

func (bba *BBA) broadcast(round uint64, req cleisthenes.Request) error {
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
			Round:   round,
			Type:    typ,
			Payload: payload,
		},
	}
	broadcastMsg := pb.Message{
		Proposer:  bba.proposer.Address.String(),
		Sender:    bba.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Epoch:     uint64(bba.epoch),
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
			return
		case req := <-bba.reqChan:
			bba.muxMessage(req.sender, req.round, req.data)
		case r := <-bba.binValueChan:
			if bba.matchWithCurrentRound(r) {
				bba.broadcastAuxOnceForRound(r)
				bba.handleDelayedAuxRequest(r)
			}
		case r := <-bba.tryoutAgreementChan:
			if bba.matchWithCurrentRound(r) {
				bba.tryoutAgreement(r)
			}
		case r := <-bba.advanceRoundChan:
			if bba.matchWithCurrentRound(r) {
				bba.advanceRound(r)
			}
		}
	}
}

func (bba *BBA) saveBvalIfNotExist(round uint64, sender cleisthenes.Member, data *BvalRequest) error {
	//_, err := bba.bvalRepo.Find(sender.Address)
	//if err != nil && !IsErrNoResult(err) {
	//	return err
	//}
	//if r != nil {
	//	return nil
	//}
	return bba.bvalRepo[round].Save(sender.Address, data)
}

func (bba *BBA) saveAuxIfNotExist(round uint64, sender cleisthenes.Member, data *AuxRequest) error {
	//_, err := bba.auxRepo.Find(sender.Address)
	//if err != nil && !IsErrNoResult(err) {
	//	return err
	//}
	//if r != nil {
	//	return nil
	//}
	return bba.auxRepo[round].Save(sender.Address, data)
}

func (bba *BBA) countBvalByValue(round uint64, val cleisthenes.Binary) int {
	bvalList := bba.convToBvalList(bba.bvalRepo[round].FindAll())
	count := 0
	for _, bval := range bvalList {
		if bval.Value == val {
			count++
		}
	}
	return count
}

func (bba *BBA) countAuxByValue(round uint64, val cleisthenes.Binary) int {
	auxList := bba.convToAuxList(bba.auxRepo[round].FindAll())
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
