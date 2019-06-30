package honeybadger

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

const initialEpoch = 0

type HoneyBadger struct {
	acsRepository *acsRepository

	memberMap     *cleisthenes.MemberMap
	txQueue       cleisthenes.TxQueue
	resultSender  cleisthenes.ResultSender
	batchReceiver cleisthenes.BatchReceiver
	acsFactory    ACSFactory

	tpk cleisthenes.Tpke

	epoch cleisthenes.Epoch

	contributionChan chan cleisthenes.Contribution
	closeChan        chan struct{}

	stopFlag        int32
	onConsensusFlag int32
}

func New(
	memberMap *cleisthenes.MemberMap,
	acsFactory ACSFactory,
	tpk cleisthenes.Tpke,
	batchReceiver cleisthenes.BatchReceiver,
	resultSender cleisthenes.ResultSender,
) *HoneyBadger {
	hb := &HoneyBadger{
		acsRepository: newACSRepository(),
		memberMap:     memberMap,
		txQueue:       cleisthenes.NewTxQueue(),
		acsFactory:    acsFactory,
		batchReceiver: batchReceiver,
		resultSender:  resultSender,

		tpk: tpk,

		epoch: initialEpoch,

		contributionChan: make(chan cleisthenes.Contribution),
		closeChan:        make(chan struct{}),
	}

	go hb.run()

	return hb
}

func (hb *HoneyBadger) HandleContribution(contribution cleisthenes.Contribution) {
	hb.contributionChan <- contribution
}

func (hb *HoneyBadger) HandleMessage(msg *pb.Message) error {
	a, err := hb.getACS(cleisthenes.Epoch(msg.Epoch))
	if err != nil {
		return err
	}

	addr, err := cleisthenes.ToAddress(msg.Sender)
	if err != nil {
		return err
	}
	member, ok := hb.memberMap.Member(addr)
	if !ok {
		return errors.New(fmt.Sprintf("member not exist in member map: %s", addr.String()))
	}

	return a.HandleMessage(member, msg)
}

func (hb *HoneyBadger) propose(contribution cleisthenes.Contribution) error {
	a, err := hb.getACS(hb.epoch)
	if err != nil {
		return err
	}

	data, err := hb.tpk.Encrypt(contribution.TxList)
	if err != nil {
		return err
	}

	return a.HandleInput(data)
}

// getACS returns ACS instance anyway. if ACS exist in repository for epoch
// then return it. otherwise create and save new ACS instance then return it
func (hb *HoneyBadger) getACS(epoch cleisthenes.Epoch) (ACS, error) {
	a, ok := hb.acsRepository.find(hb.epoch)
	if ok {
		return a, nil
	}

	a, err := hb.acsFactory.Create()
	if err != nil {
		return nil, err
	}
	if err := hb.acsRepository.save(epoch, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (hb *HoneyBadger) run() {
	for !hb.toDie() {
		select {
		case contribution := <-hb.contributionChan:
			hb.startConsensus()
			hb.propose(contribution)
		case batchMessage := <-hb.batchReceiver.Receive():
			hb.handleBatchMessage(batchMessage)
			hb.finConsensus()
		}
	}
}

func (hb *HoneyBadger) handleBatchMessage(batchMessage cleisthenes.BatchMessage) error {
	decryptedBatch := make(map[cleisthenes.Member][]byte)
	for member, encryptedTx := range batchMessage.Batch {
		tx, err := hb.tpk.Decrypt(encryptedTx)
		if err != nil {
			return err
		}
		decryptedBatch[member] = tx
	}

	hb.resultSender.Send(cleisthenes.ResultMessage{
		Epoch: batchMessage.Epoch,
		Batch: decryptedBatch,
	})
	return nil
}

func (hb *HoneyBadger) OnConsensus() bool {
	return atomic.LoadInt32(&(hb.onConsensusFlag)) == int32(1)
}

func (hb *HoneyBadger) startConsensus() {
	atomic.CompareAndSwapInt32(&hb.onConsensusFlag, int32(0), int32(1))
}

func (hb *HoneyBadger) finConsensus() {
	atomic.CompareAndSwapInt32(&hb.onConsensusFlag, int32(1), int32(0))
}

func (hb *HoneyBadger) Close() {
	if first := atomic.CompareAndSwapInt32(&hb.stopFlag, int32(0), int32(1)); !first {
		return
	}
	hb.closeChan <- struct{}{}
	<-hb.closeChan
	close(hb.closeChan)
}

func (hb *HoneyBadger) toDie() bool {
	return atomic.LoadInt32(&(hb.stopFlag)) == int32(1)
}
