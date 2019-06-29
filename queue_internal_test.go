package cleisthenes

import (
	"reflect"
	"testing"

	"github.com/DE-labtory/cleisthenes/pb"
)

type mockTransaction struct {
	name string
}

var mockTxVerifier = func(Transaction) bool {
	return true
}

type MockHoneyBadger struct {
	HandleContributionFunc func(contribution Contribution) error
	HandleMessageFunc      func(msg *pb.Message) error
}

func (hb *MockHoneyBadger) HandleContribution(contribution Contribution) error {
	return nil
}
func (hb *MockHoneyBadger) HandleMessage(msg *pb.Message) error {
	return nil
}

func TestQueue_peek(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "a",
		},
		&mockTransaction{
			name: "b",
		},
		&mockTransaction{
			name: "c",
		},
	}
	que := NewTxQueue()

	_, err := que.peek()
	if err == nil {
		t.Fatalf("test failed : peek must return error when queue is empty")
	}

	for i, input := range inputs {
		que.Push(input)
		tx, _ := que.peek()
		if tx != inputs[0] {
			t.Fatalf("test[%d] failed - Peek must return first element of queue", i)
		}
	}
}

func TestQueue_Poll(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "a",
		},
		&mockTransaction{
			name: "b",
		},
		&mockTransaction{
			name: "c",
		},
	}
	que := NewTxQueue()

	for i, input := range inputs {
		que.Push(input)
		result, _ := que.Poll()
		if result != input {
			t.Fatalf("test[%d] failed : poll must return the first element or queue", i)
		}
	}
}

func TestQueue_empty(t *testing.T) {
	que := NewTxQueue()
	if !que.empty() {
		t.Fatalf("test empty failed : after creating queue, empty() must return true")
	}

	que.Push(&mockTransaction{name: "tx"})
	if que.empty() {
		t.Fatalf("test empty failed : after pushing tx, empty() must return false")
	}

	que.Poll()
	if !que.empty() {
		t.Fatalf("test empty failed : after poll all element, empty() must return true")
	}
}

func TestQueue_len(t *testing.T) {
	que := NewTxQueue()
	for i := 1; i < 10; i++ {
		que.Push(&mockTransaction{name: "tx"})
		if que.Len() != i {
			t.Fatalf("test length failed : expected = %d, got = %d", i, que.Len())
		}
	}
}

func TestQueue_at(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "tx1",
		},
		&mockTransaction{
			name: "tx2",
		},
		&mockTransaction{
			name: "tx3",
		},
		&mockTransaction{
			name: "tx4",
		},
		&mockTransaction{
			name: "tx5",
		},
	}
	que := NewTxQueue()
	for _, input := range inputs {
		que.Push(input)
	}
	for i := 0; i < 5; i++ {
		result, _ := que.at(i)
		if result != inputs[i] {
			t.Fatalf("test[%d] failed. expected = %v, got = %v", i, inputs[i], result)
		}
	}

	_, err := que.at(10)
	if err == nil {
		t.Fatalf("test failed. index boundary error must be occured")
	}
}

func TestQueue_Push(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "a",
		},
		&mockTransaction{
			name: "b",
		},
		&mockTransaction{
			name: "c",
		},
	}
	que := NewTxQueue()

	for i, input := range inputs {
		que.Push(input)
		if result, _ := que.at(0); result != inputs[0] {
			t.Fatalf("test[%d] failed - queue's first element is invalid", i)
		}

		if result, _ := que.at(que.Len() - 1); result != input {
			t.Fatalf("test[%d] failed - queue's last element is invalid", i)
		}
	}
}

func TestDefaultTxQueueManager_AddTransaction(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "tx1",
		},
		&mockTransaction{
			name: "tx2",
		},
		&mockTransaction{
			name: "tx3",
		},
	}

	hb := &MockHoneyBadger{}
	manager := NewDefaultTxQueueManager(NewTxQueue(), hb)

	for i, input := range inputs {
		manager.AddTransaction(input, func(transaction Transaction) bool {
			return true
		})

		result, err := manager.txQueue.Poll()
		if err != nil {
			t.Fatalf("test[%d] failed : transaction is not pushed into queue", i)
		}

		if !reflect.DeepEqual(result, input) {
			t.Fatalf("test[%d] failed : transaction is not pushed in FIFO order", i)
		}
	}
}

func TestDefaultTxQueueManager_CreateBatch(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "tx1",
		},
		&mockTransaction{
			name: "tx2",
		},
		&mockTransaction{
			name: "tx3",
		},
	}
	myBatch := 10
	myNode := 8
	hb := &MockHoneyBadger{}
	manager := NewDefaultTxQueueManager(NewTxQueue(), hb)

	for _, input := range inputs {
		manager.AddTransaction(input, mockTxVerifier)
	}

	outputs, err := manager.createContribution()
	if err != nil {
		t.Fatalf("test failed : error occurred when polling transaction from queue")
	}

	checkIdx := make([]int, min(myBatch, myNode))

	for i, output := range outputs.TxList() {
		checkInclude := false
		for j, input := range inputs {
			if input == output {
				checkInclude = true

				if checkIdx[j] != 0 {
					t.Fatalf("test[%d] failed : polled duplicated transaction", i)
				}
				checkIdx[j]++
				break
			}
		}
		if checkInclude == false {
			t.Fatalf("test[%d] failed : transaction on queue is different from input", i)
		}
	}
}

func TestDefaultTxQueueManager_LoadCandidateTx(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "tx1",
		},
		&mockTransaction{
			name: "tx2",
		},
		&mockTransaction{
			name: "tx3",
		},
	}
	myBatch := 10
	hb := &MockHoneyBadger{}
	manager := NewDefaultTxQueueManager(NewTxQueue(), hb)

	for _, input := range inputs {
		manager.AddTransaction(input, mockTxVerifier)
	}

	candidates, err := manager.loadCandidateTx(min(myBatch, manager.txQueue.Len()))

	if err != nil {
		t.Fatalf("test failed : error occurred when polling transaction from queue: err: %s", err.Error())
	}

	for i, candidate := range candidates {
		checkInclude := false
		for _, input := range inputs {
			if input == candidate {
				checkInclude = true
				break
			}
		}
		if checkInclude == false {
			t.Fatalf("test[%d] failed : transaction of candidate is different from queue", i)
		}
	}
}

func TestDefaultTxQueueManager_SelectRandomTx(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			name: "tx1",
		},
		&mockTransaction{
			name: "tx2",
		},
		&mockTransaction{
			name: "tx3",
		},
		&mockTransaction{
			name: "tx4",
		},
		&mockTransaction{
			name: "tx5",
		},
	}
	myBatch := 10
	myNode := 8
	hb := &MockHoneyBadger{}
	manager := NewDefaultTxQueueManager(NewTxQueue(), hb)

	for _, input := range inputs {
		manager.AddTransaction(input, mockTxVerifier)
	}

	candidate, err := manager.loadCandidateTx(min(myBatch, manager.txQueue.Len()))

	if err != nil {
		t.Fatalf("test failed : error occurred when polling transaction from queue")
	}

	batch, cleanUp := manager.selectRandomTx(candidate, int(myBatch/myNode))
	checkIdx := make([]int, min(myBatch, myNode))

	for i, output := range batch {
		checkInclude := false
		for j, input := range inputs {
			if input == output {
				checkInclude = true

				if checkIdx[j] != 0 {
					t.Fatalf("test[%d] failed : polled duplicated transaction", i)
				}
				checkIdx[j]++
				break
			}
		}
		if checkInclude == false {
			t.Fatalf("test[%d] failed : transaction of batch is different from queue", i)
		}
	}

	cleanUp()

	if manager.txQueue.Len()+len(batch) != len(inputs) {
		t.Fatalf("test failed : cleanUp failed : some transactions missed")
	}
}
