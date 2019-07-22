package honeybadger

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/iLogger"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type Epoch struct {
	lock  sync.RWMutex
	value cleisthenes.Epoch
}

func NewEpoch(value cleisthenes.Epoch) *Epoch {
	return &Epoch{
		lock:  sync.RWMutex{},
		value: value,
	}
}

func (e *Epoch) up() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.value++
}

func (e *Epoch) val() cleisthenes.Epoch {
	e.lock.Lock()
	defer e.lock.Unlock()
	value := e.value
	return value
}

type contributionBuffer struct {
	lock  sync.RWMutex
	value []cleisthenes.Contribution
}

func newContributionBuffer() *contributionBuffer {
	return &contributionBuffer{
		lock:  sync.RWMutex{},
		value: make([]cleisthenes.Contribution, 0),
	}
}

func (cb *contributionBuffer) add(buffer cleisthenes.Contribution) {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.value = append(cb.value, buffer)
}

func (cb *contributionBuffer) one() cleisthenes.Contribution {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	buffer := cb.value[0]
	cb.value = append(cb.value[:0], cb.value[1:]...)
	return buffer
}

func (cb *contributionBuffer) empty() bool {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	if len(cb.value) != 0 {
		return false
	}
	return true
}

const initialEpoch = 0

type HoneyBadger struct {
	lock          sync.RWMutex
	acsRepository *acsRepository

	memberMap     *cleisthenes.MemberMap
	txQueue       cleisthenes.TxQueue
	resultSender  cleisthenes.ResultSender
	batchReceiver cleisthenes.BatchReceiver
	acsFactory    ACSFactory

	tpk cleisthenes.Tpke

	epoch *Epoch
	done  *cleisthenes.BinaryState

	contributionBuffer *contributionBuffer
	contributionChan   chan struct{}
	closeChan          chan struct{}

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
		lock:          sync.RWMutex{},
		acsRepository: newACSRepository(),
		memberMap:     memberMap,
		txQueue:       cleisthenes.NewTxQueue(),
		acsFactory:    acsFactory,
		batchReceiver: batchReceiver,
		resultSender:  resultSender,

		tpk: tpk,

		epoch: NewEpoch(initialEpoch),
		done:  cleisthenes.NewBinaryState(),

		contributionBuffer: newContributionBuffer(),
		contributionChan:   make(chan struct{}, 4),
		closeChan:          make(chan struct{}),
	}

	go hb.run()

	return hb
}

func (hb *HoneyBadger) HandleContribution(contribution cleisthenes.Contribution) {
	hb.contributionBuffer.add(contribution)
	if !hb.OnConsensus() {
		hb.contributionChan <- struct{}{}
	}
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
	a, err := hb.getACS(hb.epoch.val())
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
	if hb.epoch.val() > epoch+10 {
		return nil, errors.New("old epoch")
	}

	a, ok := hb.acsRepository.find(epoch)
	if ok {
		return a, nil
	}

	a, err := hb.acsFactory.Create(epoch)
	if err != nil {
		return nil, err
	}
	if err := hb.acsRepository.save(epoch, a); err != nil {
		a, _ := hb.acsRepository.find(epoch)
		return a, nil
	}
	return a, nil
}

func (hb *HoneyBadger) run() {
	for !hb.toDie() {
		select {
		case <-hb.contributionChan:
			if !hb.contributionBuffer.empty() && !hb.done.Value() {
				hb.startConsensus()
				if err := hb.propose(hb.contributionBuffer.one()); err != nil {
					fmt.Printf("error in propose : %s\n", err.Error())
				}
			}
		case batchMessage := <-hb.batchReceiver.Receive():
			if err := hb.handleBatchMessage(batchMessage); err != nil {
				iLogger.Debugf(nil, "error in handleBatchMessage : %s", err.Error())
			}
			hb.advanceEpoch()
			hb.finConsensus()
		}
	}
}

func (hb *HoneyBadger) handleBatchMessage(batchMessage cleisthenes.BatchMessage) error {
	decryptedBatch := make([]cleisthenes.AbstractTx, 0)
	for _, encryptedTx := range batchMessage.Batch {
		txList, err := hb.tpk.Decrypt(encryptedTx)
		if err != nil {
			return err
		}

		for _, tx := range txList {
			abstractTx := cleisthenes.AbstractTx(tx.(map[string]interface{}))
			decryptedBatch = append(decryptedBatch, abstractTx)
		}
	}

	hb.resultSender.Send(cleisthenes.ResultMessage{
		Epoch: batchMessage.Epoch,
		Batch: decryptedBatch,
	})
	return nil
}

func (hb *HoneyBadger) advanceEpoch() {
	hb.epoch.up()
	hb.closeOldEpoch(hb.epoch.val() - 1)
	hb.contributionChan <- struct{}{}
}

func (hb *HoneyBadger) closeOldEpoch(epoch cleisthenes.Epoch) {
	if epoch > 10 {
		acs, ok := hb.acsRepository.find(epoch - 10)
		if !ok {
			return
		}

		acs.Close()
		hb.acsRepository.delete(epoch - 10)
	}
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
