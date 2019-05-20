package cleisthenes_test

import (
	"bytes"
	"errors"
	"net"
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

func GetAvailablePort(startPort uint16) uint16 {
	portNumber := startPort
	for {
		strPortNumber := strconv.Itoa(int(portNumber))
		lis, err := net.Listen("tcp", "127.0.0.1:"+strPortNumber)

		if err == nil {
			_ = lis.Close()
			return portNumber
		}

		portNumber++
	}
}

func (h *MockHandler) ServeRequest(msg cleisthenes.Message) {
	h.ServeRequestFunc(msg)
}

func TestGrpcConnection_Send(t *testing.T) {
	cAddress := cleisthenes.Address{"127.0.0.1", GetAvailablePort(8000)}

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
	cAddress := cleisthenes.Address{"127.0.0.1", GetAvailablePort(8000)}

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
	cAddress := cleisthenes.Address{"127.0.0.1", GetAvailablePort(8000)}

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

func setUpConnection(cp *cleisthenes.ConnectionPool) error {
	cAddressList := make([]cleisthenes.Address, 3)

	for i, _ := range cAddressList {
		cAddressList[i] = cleisthenes.Address{"127.0.0.1", GetAvailablePort(8000)}
	}

	connList := make([]cleisthenes.Connection, 3)
	var err error

	for i := range connList {
		connList[i], err = cleisthenes.NewConnection(cAddressList[i], "TestConnection"+strconv.Itoa(i+1), mock.NewStreamWrapper())
		if err != nil {
			// return error when error occurred at making connection
			return err
		}
	}

	for _, conn := range connList {
		cp.Add(conn.Id(), conn)
	}

	return nil
}

func setUpHandler(textList []string, msgReceiveCheckList []bool, wg *sync.WaitGroup) (*MockHandler, error) {
	var err error

	mockHandler := NewMockHandler(nil)
	mockHandler.ServeRequestFunc = func(msg cleisthenes.Message) {
		if msg.GetRbc().Type != pb.RBC_VAL {
			err = errors.New("expected message type is " + string(pb.RBCType_name[int32(pb.RBC_VAL)]) + ", but got " + string(msg.GetRbc().Type))
		}

		checkReceived := false
		for i, text := range textList {
			if bytes.Equal(msg.GetRbc().Payload, []byte(text)) {
				msgReceiveCheckList[i] = true
				checkReceived = true
				wg.Done()
			}
		}
		if checkReceived == false {
			err = errors.New("received message payload is " + string(msg.GetRbc().Payload) + ", which is not expected value")
		}
	}

	if err != nil {
		// return error when error occurred at setting handler
		return nil, err
	}

	return mockHandler, nil
}

func handleMessage(cp *cleisthenes.ConnectionPool, textList []string, mockHandler *MockHandler) []pb.Message {
	for _, conn := range cp.GetAll() {
		conn.Handle(mockHandler)
	}

	for _, conn := range cp.GetAll() {
		go func(conn cleisthenes.Connection) {
			if err := conn.Start(); err != nil {
				conn.Close()
			}
		}(conn)
	}

	time.Sleep(3 * time.Second)

	msgList := make([]pb.Message, len(textList))
	for i, _ := range msgList {
		msgList[i] = pb.Message{
			Payload: &pb.Message_Rbc{
				Rbc: &pb.RBC{
					Payload: []byte(textList[i]),
					Type:    pb.RBC_VAL,
				},
			},
		}
	}

	return msgList
}

func executeDistributeMessage(textList []string, waitSize int) ([]bool, error) {
	cp := cleisthenes.NewConnectionPool()

	err := setUpConnection(cp)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(waitSize)

	msgReceiveCheckList := make([]bool, len(textList))

	mockHandler, err := setUpHandler(textList, msgReceiveCheckList, &wg)
	if err != nil {
		return nil, err
	}

	msgList := handleMessage(cp, textList, mockHandler)
	cp.DistributeMessage(msgList)

	// wait until receive the msg
	wg.Wait()

	return msgReceiveCheckList, nil
}

// Text case for ShareMessage with NodeNum = 3
func TestConnectionPool_ShareMessage(t *testing.T) {
	cp := cleisthenes.NewConnectionPool()
	textList := []string{"jang"}

	err := setUpConnection(cp)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	msgReceiveCheckList := make([]bool, len(textList))
	mockHandler, err := setUpHandler(textList, msgReceiveCheckList, &wg)
	if err != nil {
		t.Fatal(err)
	}

	msgList := handleMessage(cp, textList, mockHandler)
	cp.ShareMessage(msgList[0])

	// wait until receive the msg
	wg.Wait()
}

// Test case for NodeNum == MessageNum (Both of them are 3) at DistributeMessage
func TestConnectionPool_DistributeMessage_EqualNum(t *testing.T) {
	textList := []string{"jang", "gun", "hee"}
	msgReceiveCheckList, err := executeDistributeMessage(textList, 3)

	if err != nil {
		t.Fatal(err)
	}

	for i, check := range msgReceiveCheckList {
		if check == false {
			t.Fatalf("message payload %s has missed", textList[i])
		}
	}
}

// Test case for NodeNum < MessageNum (NodeNum is 3, MessageNum is 4) at DistributeMessage
func TestConnectionPool_DistributeMessage_MessageNumBigger(t *testing.T) {
	textList := []string{"jang", "gun", "hee", "kim"}
	msgReceiveCheckList, err := executeDistributeMessage(textList, 3)

	if err != nil {
		t.Fatal(err)
	}

	for i, check := range msgReceiveCheckList {
		if i == 3 {
			break
		}
		if check == false {
			t.Fatalf("message payload %s has missed", textList[i])
		}
	}
}

// Test case for NodeNum > MessageNum (NodeNum is 3, MessageNum is 2) at DistributeMessage
func TestConnectionPool_DistributeMessage_NodeNumBigger(t *testing.T) {
	textList := []string{"jang", "gun"}
	msgReceiveCheckList, err := executeDistributeMessage(textList, 2)

	if err != nil {
		t.Fatal(err)
	}

	for i, check := range msgReceiveCheckList {
		if check == false {
			t.Fatalf("message payload %s has missed", textList[i])
		}
	}
}

func TestConnectionPool_Add(t *testing.T) {
	cAddr := cleisthenes.Address{"127.0.0.1", GetAvailablePort(8000)}
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
	cAddr := cleisthenes.Address{"127.0.0.1", 8000}
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
