package cleisthenes

import (
	"math/rand"
	"time"
)

type Batch struct {
	// txs is a transaction set of batch which is polled from queue
	txs []Transaction
}

// TxList is a function returns the transaction list on batch
func (batch *Batch) TxList() []Transaction {
	return batch.txs
}

type HoneyBadger struct {
	// TODO : HoneyBadger must have ACS

	// TODO : HoneyBadger must have consensused batch

	// que have transaction FIFO memQueue.
	que queue

	// current epoch
	epoch int

	// b is maximum number of transactions in each epoch
	b int

	// n is number of node at network
	n int
}

func NewHoneyBadger(batchSize int, nodes int) *HoneyBadger {
	maxTx := batchSize

	if batchSize < nodes {
		maxTx = nodes
	}

	return &HoneyBadger{
		que:   NewQueue(),
		epoch: 0,
		b:     maxTx,
		n:     nodes,
	}
}

// Add transaction to FIFO memQueue
func (hb *HoneyBadger) AddTransaction(tx Transaction) {
	hb.que.Push(tx)
}

// Send batch to ACS
func (hb *HoneyBadger) sendBatch() {
	panic("implement me w/ test case :-)")
}

// Create batch polling random transaction in queue
func (hb *HoneyBadger) createBatch() (Batch, error) {
	candidate, err := hb.loadCandidateTx(min(hb.b, hb.que.Len()))
	if err != nil {
		return Batch{}, err
	}

	batch, cleanUp := hb.selectRandomTx(candidate, int(hb.b/hb.n))
	defer cleanUp()

	return Batch{batch}, nil
}

// loadCandidateTx is a function that returns transactions from queue
func (hb *HoneyBadger) loadCandidateTx(candSize int) ([]Transaction, error) {
	candidate := make([]Transaction, candSize)
	var err error
	for i := 0; i < candSize; i++ {
		candidate[i], err = hb.que.Poll()

		if err != nil {
			return nil, err
		}
	}
	return candidate, nil
}

// selectRandomTx is a function that returns transactions which is randomly selected from input transactions
func (hb *HoneyBadger) selectRandomTx(candidate []Transaction, selectSize int) ([]Transaction, func()) {
	batch := make([]Transaction, selectSize)

	for i := 0; i < selectSize; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(candidate))

		batch[i] = candidate[idx]
		candidate = append(candidate[:idx], candidate[idx+1:]...)
	}
	return batch, func() {
		for _, item := range candidate {
			hb.que.Push(item)
		}
	}
}

func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

// Define transaction interface.
type Transaction interface{}
