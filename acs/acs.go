package acs

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/DE-labtory/iLogger"

	"github.com/DE-labtory/cleisthenes/rbc"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
)

type request struct {
	proposer cleisthenes.Member
	sender   cleisthenes.Member
	data     *pb.Message
	err      chan error
}

type ACS struct {
	// number of network nodes
	n int
	// number of byzantine nodes which can tolerate
	f int

	epoch cleisthenes.Epoch
	owner cleisthenes.Member

	memberMap cleisthenes.MemberMap
	output    map[cleisthenes.Member][]byte
	// rbcMap has rbc instances
	rbcRepo *RBCRepository

	// bbaMap has bba instances
	bbaRepo *BBARepository

	// broadcastResult collects RBC instances' result
	broadcastResult broadcastDataMap
	// agreementResult collects BBA instances' result
	// each entry have three states: undefined, zero, one
	agreementResult binaryStateMap
	// agreementStarted saves whether BBA instances started
	// binary byzantine agreement
	agreementStarted binaryStateMap

	dec cleisthenes.BinaryState

	tmp cleisthenes.BinaryState

	reqChan       chan request
	agreementChan chan struct{}
	closeChan     chan struct{}

	stopFlag int32

	dataReceiver   cleisthenes.DataReceiver
	binaryReceiver cleisthenes.BinaryReceiver
	batchSender    cleisthenes.BatchSender

	roundReceiver cleisthenes.BinaryReceiver
}

// TODO : the coin generator must be injected outside the ACS, BBA should has it's own coin generator.
// TODO : if we consider dynamic network, change MemberMap into pointer
func New(
	n int,
	f int,
	epoch cleisthenes.Epoch,
	owner cleisthenes.Member,
	memberMap cleisthenes.MemberMap,
	dataReceiver cleisthenes.DataReceiver,
	dataSender cleisthenes.DataSender,
	binaryReceiver cleisthenes.BinaryReceiver,
	binarySender cleisthenes.BinarySender,
	batchSender cleisthenes.BatchSender,
	broadCaster cleisthenes.Broadcaster,
) (*ACS, error) {

	acs := &ACS{
		n:                n,
		f:                f,
		epoch:            epoch,
		owner:            owner,
		memberMap:        memberMap,
		rbcRepo:          NewRBCRepository(),
		bbaRepo:          NewBBARepository(),
		broadcastResult:  NewbroadcastDataMap(),
		agreementResult:  NewBinaryStateMap(),
		agreementStarted: NewBinaryStateMap(),
		// TODO : consider size of reqChan, otherwise this might cause requests to be lost
		reqChan:        make(chan request, n*n*8),
		agreementChan:  make(chan struct{}, n*8),
		closeChan:      make(chan struct{}),
		dataReceiver:   dataReceiver,
		binaryReceiver: binaryReceiver,
		batchSender:    batchSender,
	}

	for _, member := range memberMap.Members() {
		r, err := rbc.New(n, f, epoch, owner, member, broadCaster, dataSender)
		b := bba.New(n, f, epoch, owner, member, broadCaster, binarySender)
		if err != nil {
			return nil, err
		}
		acs.rbcRepo.Save(member, r)
		acs.bbaRepo.Save(member, b)

		acs.broadcastResult.set(member, []byte{})
		acs.agreementResult.set(member, *cleisthenes.NewBinaryState())
		acs.agreementStarted.set(member, *cleisthenes.NewBinaryState())
	}

	go acs.run()
	return acs, nil
}

// HandleInput receive encrypted batch from honeybadger
func (acs *ACS) HandleInput(data []byte) error {
	rbc, err := acs.rbcRepo.Find(acs.owner)
	if err != nil {
		return errors.New(fmt.Sprintf("no match rbc instance - address : %s", acs.owner.Address.String()))
	}

	return rbc.HandleInput(data)
}

func (acs *ACS) HandleMessage(sender cleisthenes.Member, msg *pb.Message) error {
	proposerAddr, err := cleisthenes.ToAddress(msg.Proposer)
	if err != nil {
		return err
	}
	proposer, ok := acs.memberMap.Member(proposerAddr)
	if !ok {
		return ErrNoMemberMatchingRequest
	}
	req := request{
		proposer: proposer,
		sender:   sender,
		data:     msg,
		err:      make(chan error),
	}

	acs.reqChan <- req
	return <-req.err
}

// Result return consensused batch to honeybadger component
func (acs *ACS) Result() cleisthenes.Batch {
	return cleisthenes.Batch{}
}

func (acs *ACS) Close() {
	for _, rbc := range acs.rbcRepo.FindAll() {
		rbc.Close()
	}

	for _, bba := range acs.bbaRepo.FindAll() {
		bba.Close()
	}

	acs.closeChan <- struct{}{}
	<-acs.closeChan
	if first := atomic.CompareAndSwapInt32(&acs.stopFlag, int32(0), int32(1)); !first {
		return
	}
	//close(acs.closeChan)
}

func (acs *ACS) muxMessage(proposer, sender cleisthenes.Member, msg *pb.Message) error {
	switch pl := msg.Payload.(type) {
	case *pb.Message_Rbc:
		return acs.handleRbcMessage(proposer, sender, pl)
	case *pb.Message_Bba:
		return acs.handleBbaMessage(proposer, sender, pl)
	default:
		return cleisthenes.ErrUndefinedRequestType
	}
}

func (acs *ACS) handleRbcMessage(proposer, sender cleisthenes.Member, msg *pb.Message_Rbc) error {

	rbc, err := acs.rbcRepo.Find(proposer)
	if err != nil {
		return errors.New(fmt.Sprintf("no match bba instance - address : %s", acs.owner.Address.String()))

	}
	return rbc.HandleMessage(sender, msg)
}

func (acs *ACS) handleBbaMessage(proposer, sender cleisthenes.Member, msg *pb.Message_Bba) error {
	bba, err := acs.bbaRepo.Find(proposer)
	if err != nil {
		return errors.New(fmt.Sprintf("no match bba instance - address : %s", acs.owner.Address.String()))

	}
	return bba.HandleMessage(sender, msg)
}

func (acs *ACS) run() {
	for !acs.toDie() {
		select {
		case <-acs.closeChan:
			acs.closeChan <- struct{}{}
			return
		case req := <-acs.reqChan:
			req.err <- acs.muxMessage(req.proposer, req.sender, req.data)
		case output := <-acs.dataReceiver.Receive():
			acs.processData(output.Member, output.Data)
			acs.tryAgreementStart(output.Member)
			acs.tryCompleteAgreement()
		case <-acs.agreementChan:
			acs.sendZeroToIdleBba()
		case output := <-acs.binaryReceiver.Receive():
			acs.processAgreement(output.Member, output.Binary)
			acs.tryCompleteAgreement()
		}
	}
}

// sendZeroToIdleBba send zero to bba instances which still do
// not receive input value
func (acs *ACS) sendZeroToIdleBba() {
	for _, member := range acs.memberMap.Members() {
		b, err := acs.bbaRepo.Find(member)
		if err != nil {
			fmt.Printf("no match bba instance - address : %s\n", member.Address.String())
		}

		if state := acs.agreementStarted.item(member); state.Undefined() && b.Idle() {
			acs.tmp.Set(true)
			if err := b.HandleInput(0, &bba.BvalRequest{Value: cleisthenes.Zero}); err != nil {
				fmt.Printf("error in HandleInput : %s", err.Error())
			}
		}
	}
}

func (acs *ACS) tryCompleteAgreement() {
	if acs.dec.Value() || acs.countDoneAgreement() != acs.agreementDoneThreshold() {
		return
	}

	agreedNodeList := make([]cleisthenes.Member, 0)
	for member, state := range acs.agreementResult.itemMap() {
		if state.Value() {
			agreedNodeList = append(agreedNodeList, member)
		}
	}

	bcResult := make(map[cleisthenes.Member][]byte)

	for _, member := range agreedNodeList {
		if data := acs.broadcastResult.item(member); data != nil && len(data) != 0 {
			bcResult[member] = data
		}
	}

	if len(agreedNodeList) == len(bcResult) && len(bcResult) != 0 {
		acs.agreementSuccess(bcResult)
	}
}

func (acs *ACS) agreementSuccess(result map[cleisthenes.Member][]byte) {
	for member, _ := range result {
		fmt.Println(member.Address.String())
	}
	iLogger.Debugf(nil, "[ACS done] epoch : %d, owner : %s\n", acs.epoch, acs.owner.Address.String())
	acs.batchSender.Send(cleisthenes.BatchMessage{
		Epoch: acs.epoch,
		Batch: result,
	})
	acs.dec.Set(true)
}

func (acs *ACS) tryAgreementStart(sender cleisthenes.Member) {
	b, err := acs.bbaRepo.Find(sender)
	if err != nil {
		fmt.Printf("no match bba instance - address : %s\n", acs.owner.Address.String())
		return
	}

	if ok := acs.agreementStarted.exist(sender); !ok {
		fmt.Printf("no match agreementStarted item - address : %s\n", sender.Address.String())
		return
	}

	state := acs.agreementStarted.item(sender)
	if state.Value() {
		fmt.Printf("already started bba - address :%s\n", sender.Address)
		return
	}

	if err := b.HandleInput(b.Round(), &bba.BvalRequest{Value: cleisthenes.One}); err != nil {
		fmt.Printf("error in HandleInput : %s\n", err.Error())
	}
	state.Set(cleisthenes.One)
	acs.agreementStarted.set(sender, state)
	if acs.tmp.Value() {
		fmt.Printf("acs try start epoch : %d, onwer : %s, proposer : %s\n", acs.epoch, acs.owner.Address.String(), sender.Address.String())
	}
	return
}

func (acs *ACS) processData(sender cleisthenes.Member, data []byte) {
	if ok := acs.broadcastResult.exist(sender); !ok {
		fmt.Printf("no match broadcastResult item - address : %s\n", sender.Address.String())
		return
	}

	if data := acs.broadcastResult.item(sender); len(data) != 0 {
		fmt.Printf("already processed data - address : %s\n", sender.Address.String())
		return
	}

	acs.broadcastResult.set(sender, data)
}

func (acs *ACS) processAgreement(sender cleisthenes.Member, bin cleisthenes.Binary) {
	if ok := acs.agreementResult.exist(sender); !ok {
		fmt.Printf("no match agreementResult item - address : %s\n", sender.Address.String())
		return
	}

	state := acs.agreementResult.item(sender)
	if state.Value() {
		fmt.Printf("already processed agreement - address : %s\n", sender.Address.String())
		return
	}

	state.Set(bin)
	acs.agreementResult.set(sender, state)
	//iLogger.Debugf(nil, "bba done epoch : %d, onwer : %s, proposer : %s\n", acs.epoch, acs.owner.Address.String(), sender.Address.String())
	if acs.countSuccessDoneAgreement() == acs.agreementThreshold() {
		acs.agreementChan <- struct{}{}
	}
}

func (acs *ACS) countSuccessDoneAgreement() int {
	cnt := 0
	for _, state := range acs.agreementResult.itemMap() {
		if state.Value() {
			cnt++
		}
	}
	return cnt
}

func (acs *ACS) countDoneAgreement() int {
	cnt := 0
	for _, state := range acs.agreementResult.itemMap() {
		if !state.Undefined() {
			cnt++
		}
	}
	return cnt
}

func (acs *ACS) agreementThreshold() int {
	return acs.n - acs.f
}

func (acs *ACS) agreementDoneThreshold() int {
	return acs.n
}

func (acs *ACS) toDie() bool {
	return atomic.LoadInt32(&(acs.stopFlag)) == int32(1)
}
