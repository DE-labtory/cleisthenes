package cleisthenes

import (
	"math/rand"
	"time"
)

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

func NewHoneyBadger(batch int, nodes int) *HoneyBadger {
	if batch < nodes {
		batch = nodes
	}

	return &HoneyBadger{
		que:   NewQueue(),
		epoch: 0,
		b:     batch,
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
func (hb *HoneyBadger) createBatch() []Transaction {
	minVal := min(hb.b, hb.que.Len())
	buf := make([]Transaction, minVal)
	pol := make([]Transaction, int(hb.b/hb.n))

	for i := 0; i < minVal; i++ {
		buf[i], _ = hb.que.Poll()
	}

	for i := 0; i < int(hb.b/hb.n); i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(minVal)

		pol[i] = buf[idx]
		buf = append(buf[:idx],buf[idx+1:]...)
		minVal--
	}

	for i := 0; i < minVal; i++ {
		hb.que.Push(buf[i])
	}

	return pol
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
