package cleisthenes

import (
	"reflect"
	"testing"
)

func TestBinaryChannel_Send(t *testing.T) {
	binChan := NewBinaryChannel(1)

	member := Member{Address: Address{Ip: "127.0.0.1", Port: 8080}}

	binChan.Send(BinaryMessage{
		Member: member,
		Binary: One,
	})

	msg := <-binChan.buffer
	if !reflect.DeepEqual(&msg.Member, &member) {
		t.Fatalf("expected member is %v, but got %v", member, msg.Member)
	}
	if msg.Binary != One {
		t.Fatalf("expected binary is %v, but got %v", One, msg.Binary)
	}
}

func TestBinaryChannel_Receive(t *testing.T) {
	binChan := NewBinaryChannel(1)

	member := Member{Address: Address{Ip: "127.0.0.1", Port: 8080}}

	binChan.buffer <- BinaryMessage{
		Member: member,
		Binary: One,
	}

	msg := <-binChan.Receive()
	if !reflect.DeepEqual(&msg.Member, &member) {
		t.Fatalf("expected member is %v, but got %v", member, msg.Member)
	}
	if msg.Binary != One {
		t.Fatalf("expected binary is %v, but got %v", One, msg.Binary)
	}
}
