package cleisthenes

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DE-labtory/iLogger"
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
	q.Lock()
	defer q.Unlock()
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

type TxValidator func(Transaction) bool

// TxQueueManager manages transaction queue. It receive transaction from client
// and TxQueueManager have its own policy to propose contribution to honeybadger
type TxQueueManager interface {
	AddTransaction(tx Transaction) error
}

type DefaultTxQueueManager struct {
	txQueue TxQueue
	hb      HoneyBadger

	stopFlag int32

	contributionSize int
	batchSize        int

	closeChan chan struct{}

	tryInterval time.Duration

	txValidator TxValidator
}

func NewDefaultTxQueueManager(
	txQueue TxQueue,
	hb HoneyBadger,
	contributionSize int,
	batchSize int,

	// tryInterval is time interval to try create contribution
	// then propose to honeybadger component
	tryInterval time.Duration,

	txVerifier TxValidator,
) *DefaultTxQueueManager {
	m := &DefaultTxQueueManager{
		txQueue:          txQueue,
		hb:               hb,
		contributionSize: contributionSize,
		batchSize:        batchSize,

		closeChan:   make(chan struct{}),
		tryInterval: tryInterval,
		txValidator: txVerifier,
	}

	go m.runContributionProposeRoutine()

	return m
}

func (m *DefaultTxQueueManager) AddTransaction(tx Transaction) error {
	if !m.txValidator(tx) {
		return errors.New(fmt.Sprintf("error invalid transaction: %v", tx))
	}
	m.txQueue.Push(tx)
	return nil
}

func (m *DefaultTxQueueManager) Close() {
	if first := atomic.CompareAndSwapInt32(&m.stopFlag, int32(0), int32(1)); !first {
		return
	}
	m.closeChan <- struct{}{}
	<-m.closeChan
}

func (m *DefaultTxQueueManager) toDie() bool {
	return atomic.LoadInt32(&(m.stopFlag)) == int32(1)
}

// runContributionProposeRoutine tries to propose contribution every its "tryInterval"
// And if honeybadger is on consensus, it waits
func (m *DefaultTxQueueManager) runContributionProposeRoutine() {
	for !m.toDie() {
		if !m.hb.OnConsensus() {
			m.tryPropose()
		}
		iLogger.Debugf(nil, "[DefaultTxQueueManager] try to propose contribution...")
		time.Sleep(m.tryInterval)
	}
}

// tryPropose create contribution and send it to honeybadger only when
// transaction queue size is larger than batch size
func (m *DefaultTxQueueManager) tryPropose() error {
	if m.txQueue.Len() < int(m.contributionSize) {
		return nil
	}

	contribution, err := m.createContribution()
	if err != nil {
		return err
	}

	m.hb.HandleContribution(contribution)
	return nil
}

// Create batch polling random transaction in queue
// One caution is that caller of this function should ensure transaction queue
// size is larger than contribution size
func (m *DefaultTxQueueManager) createContribution() (Contribution, error) {
	candidate, err := m.loadCandidateTx(min(m.batchSize, m.txQueue.Len()))
	if err != nil {
		return Contribution{}, err
	}

	return Contribution{
		TxList: m.selectRandomTx(candidate, m.contributionSize),
	}, nil
}

// loadCandidateTx is a function that returns candidate transactions which could be
// included into contribution from the queue
func (m *DefaultTxQueueManager) loadCandidateTx(candidateSize int) ([]Transaction, error) {
	candidate := make([]Transaction, candidateSize)
	var err error
	for i := 0; i < candidateSize; i++ {
		candidate[i], err = m.txQueue.Poll()

		if err != nil {
			return nil, err
		}
	}
	return candidate, nil
}

// selectRandomTx is a function that returns transactions which is randomly selected from input transactions
func (m *DefaultTxQueueManager) selectRandomTx(candidate []Transaction, selectSize int) []Transaction {
	batch := make([]Transaction, selectSize)

	for i := 0; i < selectSize; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(candidate))

		batch[i] = candidate[idx]
		candidate = append(candidate[:idx], candidate[idx+1:]...)
	}
	for _, leftover := range candidate {
		m.txQueue.Push(leftover)
	}
	return batch
}

func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}
