package bba

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/it-chain/iLogger"

	"github.com/golang/protobuf/ptypes"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

var ErrUndefinedRequestType = errors.New("unexpected request type")

type request struct {
	sender cleisthenes.Member
	data   *pb.Message_Bba
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

		reqChan:             make(chan request),
		closeChan:           make(chan struct{}),
		binValueChan:        make(chan struct{}),
		tryoutAgreementChan: make(chan struct{}),
		advanceRoundChan:    make(chan struct{}),

		broadcaster:   broadcaster,
		coinGenerator: coinGenerator,
	}
	go instance.run()
	return instance
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (bba *BBA) HandleInput(msg *pb.Message_Bba) error {
	req := request{
		sender: bba.owner,
		data:   msg,
		err:    make(chan error),
	}
	bba.reqChan <- req
	return <-req.err
}

// HandleMessage will process the given rpc message.
func (bba *BBA) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	req := request{
		sender: sender,
		data:   msg,
		err:    make(chan error),
	}
	bba.reqChan <- req
	return <-req.err
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
}

func (bba *BBA) muxMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	// TODO: save message if round is larger

	switch msg.Bba.Type {
	case pb.BBA_BVAL:
		bval := &BvalRequest{}
		if err := json.Unmarshal(msg.Bba.Payload, bval); err != nil {
			return err
		}
		return bba.handleBvalRequest(sender, bval)
	case pb.BBA_AUX:
		aux := &AuxRequest{}
		if err := json.Unmarshal(msg.Bba.Payload, aux); err != nil {
			return err
		}
		return bba.handleAuxRequest(sender, aux)
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
		bba.binValueSet.union(bval.Value)
		bba.binValueChan <- struct{}{}
		return nil
	}
	if count == bba.bvalBroadcastThreshold() && !bba.broadcastedBvalSet.exist(bval.Value) {
		bba.broadcastedBvalSet.union(bval.Value)
		return bba.broadcast(pb.BBA_BVAL, bval)
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

func (bba *BBA) broadcast(typ pb.BBAType, req cleisthenes.Request) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	bba.broadcaster.ShareMessage(pb.Message{
		Timestamp: ptypes.TimestampNow(),
		Payload: &pb.Message_Bba{
			Bba: &pb.BBA{
				Round:   bba.round,
				Type:    pb.BBA_BVAL,
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
			iLogger.Infof(nil, "action: handleMessage, type: %s, from: %s", req.data.Bba.Type, req.sender.Address.String())
			err := bba.muxMessage(req.sender, req.data)
			if err != nil {
				iLogger.Errorf(nil, "action: handleMessage, type: %s, from: %s, err=%s", req.data.Bba.Type, req.sender.Address.String(), err)
			}
			req.err <- err
		case <-bba.binValueChan:
			// TODO: block if already broadcast AUX message for this round
			iLogger.Infof(nil, "action: broadcastBinValue")
			for _, bin := range bba.binValueSet.toList() {
				if err := bba.broadcast(pb.BBA_AUX, &AuxRequest{
					Value: bin,
				}); err != nil {
					iLogger.Errorf(nil, "action: handleMessage, err=%s", err)
				}
			}
		case <-bba.tryoutAgreementChan:
			iLogger.Infof(nil, "action: tryoutAgreement")
			bba.tryoutAgreement()
		case <-bba.advanceRoundChan:
			iLogger.Infof(nil, "action: advanceRound")
			bba.advanceRound()
		}
	}
}

func (bba *BBA) saveBvalIfNotExist(sender cleisthenes.Member, data *BvalRequest) error {
	r, err := bba.bvalRepo.Find(sender.Address)
	if err != nil {
		return err
	}
	if r != nil {
		return nil
	}
	return bba.bvalRepo.Save(sender.Address, data)
}

func (bba *BBA) saveAuxIfNotExist(sender cleisthenes.Member, data *AuxRequest) error {
	r, err := bba.auxRepo.Find(sender.Address)
	if err != nil {
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
