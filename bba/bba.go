package bba

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/iLogger"

	"github.com/golang/protobuf/ptypes"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

var ErrUndefinedRequestType = errors.New("unexpected request type")

type request struct {
	sender cleisthenes.Member
	data   cleisthenes.Request
	round  uint64
	err    chan error
}

type BBA struct {
	*sync.RWMutex
	owner cleisthenes.Member
	// number of network nodes
	n int
	// number of byzantine nodes which can tolerate
	f int

	stopFlag int32
	// done is flag whether BBA is terminated or not
	done bool
	// epoch is current honeybadger epoch
	epoch uint64
	// round is value for current Binary Byzantine Agreement round
	round       uint64
	binValueSet *binarySet
	// broadcastedBvalSet is set of bval value instance has sent
	broadcastedBvalSet *binarySet
	// est is estimated value of BBA instance, dec is decision value
	est, dec *cleisthenes.BinaryState

	bvalRepo        cleisthenes.RequestRepository
	auxRepo         cleisthenes.RequestRepository
	incomingReqRepo incomingRequestRepository

	reqChan             chan request
	closeChan           chan struct{}
	binValueChan        chan struct{}
	tryoutAgreementChan chan struct{}
	advanceRoundChan    chan struct{}

	broadcaster cleisthenes.Broadcaster

	coinGenerator cleisthenes.CoinGenerator
}

func New(n int, f int, epoch uint64, owner cleisthenes.Member, broadcaster cleisthenes.Broadcaster, coinGenerator cleisthenes.CoinGenerator) *BBA {
	instance := &BBA{
		owner: owner,
		n:     n,
		f:     f,
		epoch: epoch,
		round: 1,

		binValueSet:        newBinarySet(),
		broadcastedBvalSet: newBinarySet(),

		est: cleisthenes.NewBinaryState(),
		dec: cleisthenes.NewBinaryState(),

		bvalRepo:        newBvalReqRepository(),
		auxRepo:         newAuxReqRepository(),
		incomingReqRepo: newDefaultIncomingRequestRepository(),

		reqChan:             make(chan request, n),
		closeChan:           make(chan struct{}, 1),
		binValueChan:        make(chan struct{}, 1),
		tryoutAgreementChan: make(chan struct{}, 1),
		advanceRoundChan:    make(chan struct{}, 1),

		broadcaster:   broadcaster,
		coinGenerator: coinGenerator,
	}
	go instance.run()
	return instance
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (bba *BBA) HandleInput(bvalRequest *BvalRequest) error {
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

func (bba *BBA) Result() cleisthenes.BinaryState {
	return *bba.dec
}

func (bba *BBA) Close() {
	if first := atomic.CompareAndSwapInt32(&bba.stopFlag, int32(0), int32(1)); !first {
		return
	}
	bba.closeChan <- struct{}{}
	<-bba.closeChan
	close(bba.reqChan)
}

func (bba *BBA) muxMessage(sender cleisthenes.Member, round uint64, req cleisthenes.Request) error {
	// TODO: save message if round is larger

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
	if count == bba.binValueSetThreshold() {
		iLogger.Infof(nil, "action: reached bin value set condition, from: %s", bba.owner.Address.String())
		bba.binValueSet.union(bval.Value)
		bba.binValueChan <- struct{}{}
		return nil
	}
	if count == bba.bvalBroadcastThreshold() && !bba.broadcastedBvalSet.exist(bval.Value) {
		bba.broadcastedBvalSet.union(bval.Value)
		iLogger.Infof(nil, "action: reached bval broadcast condition, from: %s", bba.owner.Address.String())
		return bba.broadcast(bval)
	}
	return nil
}

func (bba *BBA) handleAuxRequest(sender cleisthenes.Member, aux *AuxRequest) error {
	if err := bba.saveAuxIfNotExist(sender, aux); err != nil {
		return err
	}
	count := bba.countAuxByValue(aux.Value)
	if count < bba.tryoutAgreementThreshold() {
		return nil
	}
	bba.tryoutAgreementChan <- struct{}{}
	return nil
}

func (bba *BBA) tryoutAgreement() {
	if bba.done {
		return
	}
	coin := bba.coinGenerator.Coin()

	binList := bba.binValueSet.toList()
	if len(binList) == 0 {
		log.Fatalf("binary set is empty, but tried agreement")
		return
	}
	if len(binList) > 1 {
		bba.est.Set(cleisthenes.Binary(coin))
		bba.advanceRoundChan <- struct{}{}
		return
	}

	if binList[0] != cleisthenes.Binary(coin) {
		bba.est.Set(binList[0])
		bba.advanceRoundChan <- struct{}{}
		return
	}
	bba.dec.Set(binList[0])
	bba.done = true
	bba.advanceRoundChan <- struct{}{}
}

func (bba *BBA) advanceRound() {
	bba.bvalRepo = newBvalReqRepository()
	bba.auxRepo = newAuxReqRepository()
	bba.binValueSet = newBinarySet()

	bba.round++

	// TODO: handle delayed messages
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
			iLogger.Infof(nil, "action: handleMessage, type: %T, from: %s", req.data, req.sender.Address.String())
			err := bba.muxMessage(req.sender, req.round, req.data)
			if err != nil {
				iLogger.Errorf(nil, "action: handleMessage, type: %s, from: %s, err=%s", req.data, req.sender.Address.String(), err)
			}
			req.err <- err
		case <-bba.binValueChan:
			// TODO: block if already broadcast AUX message for this round
			iLogger.Infof(nil, "action: broadcastBinValue, from: %s", bba.owner.Address.String())
			for _, bin := range bba.binValueSet.toList() {
				if err := bba.broadcast(&AuxRequest{
					Value: bin,
				}); err != nil {
					iLogger.Errorf(nil, "action: handleMessage, err=%s", err)
				}
			}
		case <-bba.tryoutAgreementChan:
			iLogger.Infof(nil, "action: tryoutAgreement, from: %s", bba.owner.Address.String())
			bba.tryoutAgreement()
		case <-bba.advanceRoundChan:
			iLogger.Infof(nil, "action: advanceRound, from: %s", bba.owner.Address.String())
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
