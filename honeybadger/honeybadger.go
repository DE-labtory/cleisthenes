package honeybadger

import (
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type batchTransport struct {
	batch cleisthenes.Batch
	err   chan error
}

type HoneyBadger struct {
	acsMap        ACSMap
	txQueue       cleisthenes.TxQueue
	batchReceiver cleisthenes.BatchReceiver

	epoch int

	batchChan chan batchTransport
	closeChan chan struct{}

	stopFlag int32
}

func New() *HoneyBadger {
	return &HoneyBadger{
		txQueue: cleisthenes.NewTxQueue(),
		epoch:   0,
	}
}

func (hb *HoneyBadger) HandleBatch(batch cleisthenes.Batch) error {
	transporter := batchTransport{
		batch: batch,
		err:   make(chan error),
	}
	hb.batchChan <- transporter
	return <-transporter.err
}

func (hb *HoneyBadger) HandleMessage(msg *pb.Message) error {
	return nil
}

func (hb *HoneyBadger) propose(batch cleisthenes.Batch) error {
	return nil
}

func (hb *HoneyBadger) run() {
	for !hb.toDie() {
		select {
		case transporter := <-hb.batchChan:
			transporter.err <- hb.propose(transporter.batch)
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
