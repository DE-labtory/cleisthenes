package cleisthenes_test

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestConnectionPool_Broadcast(t *testing.T) {
	cp := cleisthenes.NewConnectionPool()

	cAddress1 := cleisthenes.Address{"127.0.0.1", 8080}
	cAddress2 := cleisthenes.Address{"127.0.0.1", 8081}

	mockStreamWrapper1 := mock.NewStreamWrapper()
	conn1, err1 := cleisthenes.NewConnection(cAddress1, "TestConnection", mockStreamWrapper1)
	if err1 != nil {
		t.Fatal(err1)
	}

	mockStreamWrapper2 := mock.NewStreamWrapper()
	conn2, err2 := cleisthenes.NewConnection(cAddress2, "TestConnection2", mockStreamWrapper2)
	if err2 != nil {
		t.Fatal(err2)
	}

	cp.Add(conn1.Id(), conn1)
	cp.Add(conn2.Id(), conn2)

	var wg sync.WaitGroup
	wg.Add(2)

	mockHandler := NewMockHandler(nil)
	mockHandler.ServeRequestFunc = func(msg cleisthenes.Message) {
		if msg.GetRbc().Type != pb.RBC_VAL {
			t.Fatalf("expected message type is %s, but got %s", pb.RBCType_name[int32(pb.RBC_VAL)], msg.GetRbc().Type)
		}
		if !bytes.Equal(msg.GetRbc().Payload, []byte("jang")) {
			t.Fatalf("expected message payload is %s, but got %s", "jang", string(msg.GetRbc().Payload))
		}
		wg.Done()
	}
	conn1.Handle(mockHandler)
	conn2.Handle(mockHandler)

	go func() {
		if err := conn1.Start(); err != nil {
			conn1.Close()
		}
	}()

	go func() {
		if err := conn2.Start(); err != nil {
			conn2.Close()
		}
	}()

	time.Sleep(3 * time.Second)

	msg := pb.Message{
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload: []byte("jang"),
				Type:    pb.RBC_VAL,
			},
		},
	}

	cp.Broadcast(msg)

	// wait until receive the msg
	wg.Wait()
}

func TestConnectionPool_Add(t *testing.T) {
	cAddr := cleisthenes.Address{"127.0.0.1", 8080}
	mockStreamWrapper := mock.NewStreamWrapper()
	cp := cleisthenes.NewConnectionPool()

	connNum := 5
	connList := make([]cleisthenes.Connection, connNum)
	var err error

	for i, _ := range connList {
		connId := "TestConnection" + strconv.Itoa(i+1)
		connList[i], err = cleisthenes.NewConnection(cAddr, connId, mockStreamWrapper)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, conn := range connList {
		cp.Add(conn.Id(), conn)
	}

	if connNum != len(cp.GetAll()) {
		t.Fatalf("expected connection length is %d, but got %d", connNum, len(cp.GetAll()))
	}

	for i, getConn := range cp.GetAll() {
		checkInclude := false
		for _, inputConn := range connList {
			if reflect.DeepEqual(inputConn, getConn) {
				checkInclude = true
				break
			}
		}
		if checkInclude == false {
			t.Fatalf("test[%d] failed : Connection from connection pool is not a value of input", i)
		}
	}
}

func TestConnectionPool_Remove(t *testing.T) {
	cAddr := cleisthenes.Address{"127.0.0.1", 8080}
	mockStreamWrapper := mock.NewStreamWrapper()
	cp := cleisthenes.NewConnectionPool()

	connNum := 5
	connList := make([]cleisthenes.Connection, connNum)
	var err error

	for i, _ := range connList {
		connId := "TestConnection" + strconv.Itoa(i+1)
		connList[i], err = cleisthenes.NewConnection(cAddr, connId, mockStreamWrapper)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, conn := range connList {
		cp.Add(conn.Id(), conn)
	}

	deleteNum := 2
	deleteConnList := make([]cleisthenes.ConnId, deleteNum)

	for i, _ := range deleteConnList {
		deleteConnList[i] = connList[i].Id()
	}

	for _, deleteConnId := range deleteConnList {
		cp.Remove(deleteConnId)
	}

	if connNum-deleteNum != len(cp.GetAll()) {
		t.Fatalf("expected connection length is %d, but got %d", connNum-deleteNum, len(cp.GetAll()))
	}

	for _, getConn := range cp.GetAll() {
		checkInclude := true
		for _, deleteConnId := range deleteConnList {
			if reflect.DeepEqual(deleteConnId, getConn) {
				checkInclude = false
				break
			}
		}
		if checkInclude == false {
			t.Fatalf("connection from connection pool is removed connection: error occurred when removing connection pool")
		}
	}
}
