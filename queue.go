package cleisthenes

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type TxQueue interface {
	Push(tx Transaction)
	Poll() (Transaction, error)
	Len() int
}

// memQueue defines transaction FIFO memQueue
type MemTxQueue struct {
	txs []Transaction
	sync.RWMutex
}

// EmptyQueErr is for calling peek(), poll() when memQueue is empty.
type emptyQueErr struct{}

func (e *emptyQueErr) Error() string {
	return "memQueue is empty."
}

// indexBoundaryErr is for calling indexOf
type indexBoundaryErr struct {
	queSize int
	want    int
}

func (e *indexBoundaryErr) Error() string {
	return fmt.Sprintf("index is larger than memQueue size.\n"+
		"memQueue size : %d, you want : %d", e.queSize, e.want)
}

func NewTxQueue() *MemTxQueue {
	return &MemTxQueue{
		txs: []Transaction{},
	}
}

// empty checks whether memQueue is empty or not.
func (q *MemTxQueue) empty() bool {
	return len(q.txs) == 0
}

// peek returns first element of memQueue, but not erase it.
func (q *MemTxQueue) peek() (Transaction, error) {
	if q.empty() {
		return nil, &emptyQueErr{}
	}

	return q.txs[0], nil
}

// Poll returns first element of memQueue, and erase it.
func (q *MemTxQueue) Poll() (Transaction, error) {
	q.Lock()
	defer q.Unlock()

	ret, err := q.peek()
	if err != nil {
		return nil, err
	}

	q.txs = q.txs[1:]
	return ret, nil
}

// len returns size of memQueue
func (q *MemTxQueue) Len() int {
	return len(q.txs)
}

// at returns element of index in memQueue
func (q *MemTxQueue) at(index int) (Transaction, error) {
	if index >= q.Len() {
		return nil, &indexBoundaryErr{
			queSize: q.Len(),
			want:    index,
		}
	}
	return q.txs[index], nil
}

// Push adds transaction to memQueue.
func (q *MemTxQueue) Push(tx Transaction) {
	q.Lock()
	defer q.Unlock()

	q.txs = append(q.txs, tx)
}

type TxQueueManager interface {
	AddTransaction(tx Transaction)
}

type DefaultTxQueueManager struct {
	txQueue TxQueue
	hb      HoneyBadger
}

func NewDefaultTxQueueManager(txQueue TxQueue, hb HoneyBadger) *DefaultTxQueueManager {
	return &DefaultTxQueueManager{
		txQueue: txQueue,
		hb:      hb,
	}
}

func (m *DefaultTxQueueManager) ReceiveBatch(batch Batch) {
	// send consensused batch to application

	// follow default policy then create batch
}

func (m *DefaultTxQueueManager) AddTransaction(tx Transaction) {
	// validate transaction

	m.txQueue.Push(tx)
}

// Create batch polling random transaction in queue
func (m *DefaultTxQueueManager) createBatch(n, b int) (Batch, error) {
	candidate, err := m.loadCandidateTx(min(b, m.txQueue.Len()))
	if err != nil {
		return Batch{}, err
	}

	batch, cleanUp := m.selectRandomTx(candidate, int(b/n))
	defer cleanUp()

	return Batch{txList: batch}, nil
}

// loadCandidateTx is a function that returns transactions from queue
func (m *DefaultTxQueueManager) loadCandidateTx(candSize int) ([]Transaction, error) {
	candidate := make([]Transaction, candSize)
	var err error
	for i := 0; i < candSize; i++ {
		candidate[i], err = m.txQueue.Poll()

		if err != nil {
			return nil, err
		}
	}
	return candidate, nil
}

// selectRandomTx is a function that returns transactions which is randomly selected from input transactions
func (m *DefaultTxQueueManager) selectRandomTx(candidate []Transaction, selectSize int) ([]Transaction, func()) {
	batch := make([]Transaction, selectSize)

	for i := 0; i < selectSize; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(candidate))

		batch[i] = candidate[idx]
		candidate = append(candidate[:idx], candidate[idx+1:]...)
	}
	return batch, func() {
		for _, item := range candidate {
			m.txQueue.Push(item)
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
