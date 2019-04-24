package cleisthenes

import "testing"

type mockTransaction struct {
	name string
}

func TestQueue_peek(t *testing.T) {
	inputs := []Transaction{
		&mockTransaction{
			"a",
		},
		&mockTransaction{
			name: "b",
		},
		&mockTransaction{
			name: "c",
		},
	}
	que := NewQueue()

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
			"a",
		},
		&mockTransaction{
			name: "b",
		},
		&mockTransaction{
			name: "c",
		},
	}
	que := NewQueue()

	for i, input := range inputs {
		que.Push(input)
		result, _ := que.Poll()
		if result != input {
			t.Fatalf("test[%d] failed : poll must return the first element or queue", i)
		}
	}
}

func TestQueue_empty(t *testing.T) {
	que := NewQueue()
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
	que := NewQueue()
	for i := 1; i < 10; i++ {
		que.Push(&mockTransaction{name: "tx"})
		if que.len() != i {
			t.Fatalf("test length failed : expected = %d, got = %d", i, que.len())
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
	que := NewQueue()
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
			"a",
		},
		&mockTransaction{
			name: "b",
		},
		&mockTransaction{
			name: "c",
		},
	}
	que := NewQueue()

	for i, input := range inputs {
		que.Push(input)
		if result, _ := que.at(0); result != inputs[0] {
			t.Fatalf("test[%d] failed - queue's first element is invalid", i)
		}

		if result, _ := que.at(que.len() - 1); result != input {
			t.Fatalf("test[%d] failed - queue's last element is invalid", i)
		}
	}
}
