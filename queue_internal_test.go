package cleisthenes

import (
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes/pb"
)

type mockTransaction struct {
	name string
}

var mockTxVerifier = func(Transaction) bool {
	return true
}

type MockHoneyBadger struct {
	HandleContributionFunc func(contribution Contribution)
	HandleMessageFunc      func(msg *pb.Message) error
	OnConsensusFunc        func() bool
}

func (hb *MockHoneyBadger) HandleContribution(contribution Contribution) {
	hb.HandleContributionFunc(contribution)
}
func (hb *MockHoneyBadger) HandleMessage(msg *pb.Message) error {
	return nil
}

func (hb *MockHoneyBadger) OnConsensus() bool {
	return hb.OnConsensusFunc()
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

func TestDefaultTxQueueManager_AddTransactionAndProposeContribution(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)

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

	batchSize := 24
	nodeNum := 8

	hb := &MockHoneyBadger{}
	hb.HandleContributionFunc = func(contribution Contribution) {
		if len(contribution.TxList) != len(inputs) {
			t.Fatalf("expected contribution tx list is %d, but got %d", len(inputs), len(contribution.TxList))
		}
		for _, tx := range contribution.TxList {
			included := false

			name := tx.(*mockTransaction).name
			for _, candidate := range inputs {
				if name == candidate.(*mockTransaction).name {
					included = true
				}
			}

			if !included {
				t.Fatalf("%s is not included in contribution", name)
			}
		}
		wg.Done()
	}
	hb.OnConsensusFunc = func() bool {
		return false
	}

	txValidator := func(tx Transaction) bool { return true }
	manager := NewDefaultTxQueueManager(
		NewTxQueue(),
		hb,
		batchSize/nodeNum,
		batchSize,
		time.Second,
		txValidator,
	)

	for _, input := range inputs {
		manager.AddTransaction(input)
	}

	wg.Wait()
}
