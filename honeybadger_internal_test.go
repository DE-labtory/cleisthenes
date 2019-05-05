package cleisthenes

import (
	"reflect"
	"testing"
)

func TestAddTransaction(t *testing.T) {
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
	hb := NewHoneyBadger(1,1)

	for i, input := range inputs {
		hb.AddTransaction(input)

		result, err := hb.que.Poll()
		if err != nil{
			t.Fatalf("test[%d] failed : transaction is not pushed into queue", i)
		}

		if !reflect.DeepEqual(result,input) {
			t.Fatalf("test[%d] failed : transaction is not pushed in FIFO order", i)
		}
	}
}

func TestSendBatch(t *testing.T) {

}

func TestCreateBatch(t *testing.T) {
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
	hb := NewHoneyBadger(myBatch,myNode)

	for _, input := range inputs {
		hb.AddTransaction(input)
	}

	outputs := hb.createBatch()
	checkIdx := make([]int, min(myBatch,myNode))

	for i, output := range outputs {
		checkInclude := false
		for idx,input := range inputs {
			if input == output {
				checkInclude = true

				if checkIdx[idx] != 0 {
					t.Fatalf("test[%d] failed : polled duplicated transaction", i)
				}
				checkIdx[idx]++
				break
			}
		}
		if checkInclude == false{
			t.Fatalf("test[%d] failed : transaction on queue is different from input", i)
		}
	}
}
