package cleisthenes

import (
	"fmt"
	"sync"
)

type queue interface {
	Push()
	Poll()
}

// memQueue defines transaction FIFO memQueue
type memQueue struct {
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

func NewQueue() *memQueue {
	return &memQueue{
		txs: []Transaction{},
	}
}

// empty checks whether memQueue is empty or not.
func (q *memQueue) empty() bool {
	return len(q.txs) == 0
}

// peek returns first element of memQueue, but not erase it.
func (q *memQueue) peek() (Transaction, error) {
	if q.empty() {
		return nil, &emptyQueErr{}
	}

	return q.txs[0], nil
}

// Poll returns first element of memQueue, and erase it.
func (q *memQueue) Poll() (Transaction, error) {
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
func (q *memQueue) len() int {
	return len(q.txs)
}

// at returns element of index in memQueue
func (q *memQueue) at(index int) (Transaction, error) {
	if index >= q.len() {
		return nil, &indexBoundaryErr{
			queSize: q.len(),
			want:    index,
		}
	}
	return q.txs[index], nil
}

// Push adds transaction to memQueue.
func (q *memQueue) Push(tx Transaction) {
	q.Lock()
	defer q.Unlock()

	q.txs = append(q.txs, tx)
}
