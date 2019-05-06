package cleisthenes_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/test/mock"
)

type MockHandler struct {
	done             chan struct{}
	ServeRequestFunc func(msg cleisthenes.Message)
}

func NewMockHandler(done chan struct{}) *MockHandler {
	return &MockHandler{
		done: done,
	}
}

func (h *MockHandler) ServeRequest(msg cleisthenes.Message) {
	h.ServeRequestFunc(msg)
}

func TestGrpcConnection_Send(t *testing.T) {
	cAddress := cleisthenes.Address{"127.0.0.1", 8080}

	mockStreamWrapper := mock.NewStreamWrapper()
	conn, err := cleisthenes.NewConnection(cAddress, "127.0.0.1:8081", mockStreamWrapper)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{}, 1)

	mockHandler := NewMockHandler(done)
	mockHandler.ServeRequestFunc = func(msg cleisthenes.Message) {
		if msg.GetRbc().Type != pb.RBC_VAL {
			t.Fatalf("expected message type is %s, but got %s", pb.RBCType_name[int32(pb.RBC_VAL)], msg.GetRbc().Type)
		}
		if !bytes.Equal(msg.GetRbc().Payload, []byte("kim")) {
			t.Fatalf("expected message payload is %s, but got %s", "kim", string(msg.GetRbc().Payload))
		}
		done <- struct{}{}
	}
	conn.Handle(mockHandler)

	go func() {
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	}()

	msg := pb.Message{
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload: []byte("kim"),
				Type:    pb.RBC_VAL,
			},
		},
	}

	conn.Send(msg, nil, nil)

	// wait until receive the msg
	<-done
}

func TestGrpcConnection_GetIP(t *testing.T) {
	cAddress := cleisthenes.Address{"127.0.0.1", 8080}

	mockStreamWrapper := mock.NewStreamWrapper()

	conn, err := cleisthenes.NewConnection(cAddress, "127.0.0.1:8081", mockStreamWrapper)
	if err != nil {
		t.Fatal(err)
	}

	address := conn.Ip()

	if strings.Compare(cAddress.Ip, address.Ip) != 0 || cAddress.Port != address.Port {
		t.Fatal("not equal address")
	}
}

func TestGrpcConnection_GetID(t *testing.T) {
	id := "someNetworkID"

	mockStreamWrapper := mock.NewStreamWrapper()

	conn, err := cleisthenes.NewConnection(cleisthenes.Address{}, id, mockStreamWrapper)
	if err != nil {
		t.Fatal(err)
	}

	connId := conn.Id()

	if strings.Compare(id, connId) != 0 {
		t.Fatal("not equal id")
	}
}

func TestGrpcConnection_Close(t *testing.T) {
	cAddress := cleisthenes.Address{"127.0.0.1", 8080}

	wg := sync.WaitGroup{}
	wg.Add(1)

	mockStreamWrapper := mock.NewStreamWrapper()

	conn, err := cleisthenes.NewConnection(cAddress, "127.0.0.1:8081", mockStreamWrapper)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		<-mockStreamWrapper.CloseChan
		wg.Done()
	}()

	go func() {
		if err := conn.Start(); err != nil {
			t.Fatal(err)
		}
	}()

	conn.Close()
	wg.Wait()
}
