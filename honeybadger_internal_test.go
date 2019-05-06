package cleisthenes

import (
	"reflect"
	"testing"
)

func TestHB_AddTransaction(t *testing.T) {
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
	hb := NewHoneyBadger(1, 1)

	for i, input := range inputs {
		hb.AddTransaction(input)

		result, err := hb.que.Poll()
		if err != nil {
			t.Fatalf("test[%d] failed : transaction is not pushed into queue", i)
		}

		if !reflect.DeepEqual(result, input) {
			t.Fatalf("test[%d] failed : transaction is not pushed in FIFO order", i)
		}
	}
}

func TestHB_CreateBatch(t *testing.T) {
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
	hb := NewHoneyBadger(myBatch, myNode)

	for _, input := range inputs {
		hb.AddTransaction(input)
	}

	outputs, err := hb.createBatch()
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

func TestHB_LoadCandidateTx(t *testing.T) {
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
	hb := NewHoneyBadger(myBatch, myNode)

	for _, input := range inputs {
		hb.AddTransaction(input)
	}

	candidates, err := hb.loadCandidateTx(min(hb.b, hb.que.Len()))

	if err != nil {
		t.Fatalf("test failed : error occurred when polling transaction from queue")
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

func TestHB_SelectRandomTx(t *testing.T) {
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
	hb := NewHoneyBadger(myBatch, myNode)

	for _, input := range inputs {
		hb.AddTransaction(input)
	}

	candidate, err := hb.loadCandidateTx(min(hb.b, hb.que.Len()))

	if err != nil {
		t.Fatalf("test failed : error occurred when polling transaction from queue")
	}

	batch, cleanUp := hb.selectRandomTx(candidate, int(hb.b/hb.n))
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

	if hb.que.Len()+len(batch) != len(inputs) {
		t.Fatalf("test failed : cleanUp failed : some transactions missed")
	}
}
