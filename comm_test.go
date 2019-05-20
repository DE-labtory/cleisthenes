package cleisthenes_test

import (
	"bytes"
	"testing"

	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type mockHandler struct {
	done             chan<- struct{}
	ServeRequestFunc func(msg cleisthenes.Message)
}

func newMockHandler(done chan<- struct{}) *mockHandler {
	return &mockHandler{
		done: done,
	}
}

func (h *mockHandler) ServeRequest(msg cleisthenes.Message) {
	h.ServeRequestFunc(msg)
}

func TestGrpcServer(t *testing.T) {
	//
	// setup mock handler
	//
	done := make(chan struct{})
	handler := newMockHandler(done)
	handler.ServeRequestFunc = func(msg cleisthenes.Message) {
		if msg.GetRbc().Type != pb.RBC_VAL {
			t.Fatalf("expected message type is %s, but got %s", pb.RBCType_name[int32(pb.RBC_VAL)], msg.GetRbc().Type)
		}
		if !bytes.Equal(msg.GetRbc().Payload, []byte("kim")) {
			t.Fatalf("expected message payload is %s, but got %s", "kim", string(msg.GetRbc().Payload))
		}
		t.Log("handler handles message successfully")
		done <- struct{}{}
	}

	//
	// create new grpc server
	//
	onConnection := func(conn cleisthenes.Connection) {
		t.Log("[server] on connection")
		conn.Handle(handler)
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	}
	availablePort := GetAvailablePort(8000)
	server := cleisthenes.NewServer(cleisthenes.Address{Ip: "127.0.0.1", Port: availablePort})
	server.OnConn(onConnection)
	go server.Listen()

	t.Log("sleep 1 sec for bootstrapping grpc server …")
	time.Sleep(1 * time.Second)
	//
	// create new grpc client
	//
	cli := cleisthenes.NewClient()
	conn, err := cli.Dial(cleisthenes.DialOpts{
		Addr: cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: availablePort,
		},
		Timeout: cleisthenes.DefaultDialTimeout,
	})
	if err != nil {
		t.Fatalf("dial failed with error: %s", err.Error())
	}

	// client start its connection
	go func() {
		t.Log("[client] connection start !")
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	}()

	// send message
	conn.Send(pb.Message{
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload: []byte("kim"),
				Type:    pb.RBC_VAL,
			},
		},
	}, nil, nil)

	t.Log("waiting for handler handles message ...")
	<-done
	t.Log("handler task done")
}
