package cleisthenes

import (
	"testing"

	"github.com/DE-labtory/cleisthenes/pb"
)

type MockStreamWrapper struct {
}

func (c *MockStreamWrapper) Send(msg *pb.Message) error {
	return nil
}

func (c *MockStreamWrapper) Recv() (*pb.Message, error) {
	return nil, nil
}

func (c *MockStreamWrapper) Close() {
}

func (c *MockStreamWrapper) GetStream() Stream {
	return nil
}

func TestConnectionPool_Add(t *testing.T) {
	p := NewConnectionPool()
	connection := &MockStreamWrapper{}
	conn, _ := NewConnection(Address{"127.0.0.1", 8080}, "Connection", connection)
	p.Add("Connection1", conn)
	for r, _ := range p.connMap {
		if r != "Connection1" {
			t.Fatal("Connection is not added!")
		}
	}
	if len(p.connMap) != 1 {
		t.Fatal("Connection is not added!")
	}

}
func TestConnectionPool_Remove(t *testing.T) {
	p := NewConnectionPool()
	connection := &MockStreamWrapper{}
	conn, _ := NewConnection(Address{"127.0.0.1", 8080}, "Connection", connection)
	p.Add("Connection1", conn)
	p.Add("Connection2", conn)
	p.Add("Connection3", conn)
	p.Remove("Connection1")
	for r, _ := range p.connMap {
		if r == "Connection1" {
			t.Fatal("Connection is not deleted!")
		}

	}
	if len(p.connMap) != 2 {
		t.Fatal("Connection is not deleted!")
	}

}
func TestConnectionPool_Broadcast(t *testing.T) {
	p := NewConnectionPool()
	connection := &MockStreamWrapper{}
	conn, _ := NewConnection(Address{"127.0.0.1", 8080}, "Connection", connection)
	p.Add("Connection1", conn)
	p.Add("Connection2", conn)
	p.Add("Connection3", conn)

	msg := pb.Message{
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload: []byte("kim"),
				Type:    pb.RBC_VAL,
			},
		},
	}
	p.Broadcast(msg)
	//if (connection.Recv()==(*msg ,nil){}

}
