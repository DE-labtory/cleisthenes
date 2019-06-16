package cleisthenes

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDataChannel_Send(t *testing.T) {
	dataChan := NewDataChannel(1)

	member := Member{Address: Address{Ip: "127.0.0.1", Port: 8080}}

	data := []byte("data")
	dataChan.Send(DataMessage{
		Member: member,
		Data:   data,
	})

	msg := <-dataChan.buffer
	if !reflect.DeepEqual(&msg.Member, &member) {
		t.Fatalf("expected member is %v, but got %v", member, msg.Member)
	}
	if !bytes.Equal(msg.Data, data) {
		t.Fatalf("expected binary is %v, but got %v", data, msg.Data)
	}
}

func TestDataChannel_Receive(t *testing.T) {
	dataChan := NewDataChannel(1)

	member := Member{Address: Address{Ip: "127.0.0.1", Port: 8080}}

	data := []byte("data")
	dataChan.buffer <- DataMessage{
		Member: member,
		Data:   data,
	}

	msg := <-dataChan.Receive()
	if !reflect.DeepEqual(&msg.Member, &member) {
		t.Fatalf("expected member is %v, but got %v", member, msg.Member)
	}
	if !bytes.Equal(msg.Data, data) {
		t.Fatalf("expected binary is %v, but got %v", data, msg.Data)
	}
}
