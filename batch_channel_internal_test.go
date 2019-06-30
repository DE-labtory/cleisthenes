package cleisthenes

import (
	"reflect"
	"testing"
)

func TestBatchChannel_Send(t *testing.T) {
	batchChan := NewBatchChannel(1)

	member := Member{Address: Address{Ip: "127.0.0.1", Port: 8080}}
	batch := map[Member][]byte{
		member: []byte("batch data"),
	}

	batchChan.Send(BatchMessage{
		Batch: batch,
	})

	msg := <-batchChan.buffer
	if !reflect.DeepEqual(msg.Batch, batch) {
		t.Fatalf("expected batch size is %d, but got %d", len(batch), len(msg.Batch))
	}
}

func TestBatchChannel_Receive(t *testing.T) {
	batchChan := NewBatchChannel(1)

	member := Member{Address: Address{Ip: "127.0.0.1", Port: 8080}}
	batch := map[Member][]byte{
		member: []byte("batch data"),
	}

	batchChan.buffer <- BatchMessage{
		Batch: batch,
	}

	msg := <-batchChan.Receive()
	if !reflect.DeepEqual(msg.Batch, batch) {
		t.Fatalf("expected batch size is %d, but got %d", len(batch), len(msg.Batch))
	}
}
