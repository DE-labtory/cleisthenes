package honeybadger

import (
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type batchTransport struct {
	contribution cleisthenes.Contribution
	err          chan error
}

type HoneyBadger struct {
	acsRepository *acsRepository

	memberMap     *cleisthenes.MemberMap
	txQueue       cleisthenes.TxQueue
	batchSender   cleisthenes.BatchSender
	batchReceiver cleisthenes.BatchReceiver
	acsFactory    acsFactory

	tpke cleisthenes.Tpke

	epoch cleisthenes.Epoch

	batchChan chan batchTransport
	closeChan chan struct{}

	stopFlag int32
}

func New(
//memberMap *cleisthenes.MemberMap,
) *HoneyBadger {
	return &HoneyBadger{
		acsRepository: newACSRepository(),
		txQueue:       cleisthenes.NewTxQueue(),
		//memberMap:memberMap,
		epoch: 0,
	}
}

func (hb *HoneyBadger) HandleContribution(contribution cleisthenes.Contribution) error {
	transporter := batchTransport{
		contribution: contribution,
		err:          make(chan error),
	}
	hb.batchChan <- transporter
	return <-transporter.err
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
	member := hb.memberMap.Member(addr)
	// TODO: check member exist

	return a.HandleMessage(member, msg)
}

func (hb *HoneyBadger) propose(contribution cleisthenes.Contribution) error {
	a, err := hb.getACS(hb.epoch)
	if err != nil {
		return err
	}

	data, err := hb.tpke.Encrypt(contribution)
	if err != nil {
		return err
	}

	return a.HandleInput(data)
}

func (hb *HoneyBadger) getACS(epoch cleisthenes.Epoch) (ACS, error) {
	a, ok := hb.acsRepository.find(hb.epoch)
	if ok {
		return a, nil
	}

	a, err := hb.acsFactory.create()
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
		case transporter := <-hb.batchChan:
			transporter.err <- hb.propose(transporter.contribution)
		case batchMessage := <-hb.batchReceiver.Receive():
			hb.batchSender.Send(batchMessage)
		}
	}
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
