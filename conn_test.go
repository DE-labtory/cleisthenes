package cleisthenes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/DE-labtory/cleisthenes/test/util"

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
	cAddress := cleisthenes.Address{"127.0.0.1", util.GetAvailablePort(8000)}

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
	cAddress := cleisthenes.Address{"127.0.0.1", util.GetAvailablePort(8000)}

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
	cAddress := cleisthenes.Address{"127.0.0.1", util.GetAvailablePort(8000)}

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

type samplePayload struct {
	Data string
	Id   int
}

type messageRecvChecker struct {
	lock      sync.RWMutex
	checkList []bool
}

func newMessageRecvChecker(checkSize int) *messageRecvChecker {
	return &messageRecvChecker{
		lock:      sync.RWMutex{},
		checkList: make([]bool, checkSize),
	}
}

func (m *messageRecvChecker) check(i int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.checkList[i] = true
}

func (m *messageRecvChecker) result() []bool {
	return m.checkList
}

type messageRecvCounter struct {
	lock    sync.RWMutex
	counter int
}

func newMessageRecvCounter() *messageRecvCounter {
	return &messageRecvCounter{
		lock: sync.RWMutex{},
	}
}

func (m *messageRecvCounter) count() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.counter++
}

func (m *messageRecvCounter) result() int {
	return m.counter
}

type connectionTester struct {
	connListSize   int
	msgRecvChecker *messageRecvChecker
	msgRecvCounter *messageRecvCounter
}

func newConnectionTester(connListSize int) *connectionTester {
	return &connectionTester{
		connListSize:   connListSize,
		msgRecvChecker: newMessageRecvChecker(connListSize),
		msgRecvCounter: newMessageRecvCounter(),
	}
}

func (c *connectionTester) setupConnectionPool() (*cleisthenes.ConnectionPool, error) {
	connPool := cleisthenes.NewConnectionPool()
	for i := 0; i < c.connListSize; i++ {
		addr := cleisthenes.Address{"127.0.0.1", uint16(8000 + i)}
		conn, err := cleisthenes.NewConnection(addr, "TestConnection"+strconv.Itoa(i+1), mock.NewStreamWrapper())
		if err != nil {
			return nil, err
		}

		connPool.Add(addr, conn)
	}
	return connPool, nil
}

func (c *connectionTester) setupHandler(servRequestFunc func(message cleisthenes.Message)) *MockHandler {
	mockHandler := NewMockHandler(nil)
	mockHandler.ServeRequestFunc = servRequestFunc
	return mockHandler
}

func (c *connectionTester) run(connPool *cleisthenes.ConnectionPool, handler *MockHandler) {
	for _, conn := range connPool.GetAll() {
		conn.Handle(handler)
	}

	for _, conn := range connPool.GetAll() {
		go func(conn cleisthenes.Connection) {
			if err := conn.Start(); err != nil {
				conn.Close()
			}
		}(conn)
	}
}

func (c *connectionTester) checkResult() []bool {
	return c.msgRecvChecker.result()
}

func (c *connectionTester) countResult() int {
	return c.msgRecvCounter.result()
}

func executeDistributeMessage(t *testing.T, textList []string, connListSize int) []bool {
	wg := &sync.WaitGroup{}
	wg.Add(int(math.Min(float64(len(textList)), float64(connListSize))))

	tester := newConnectionTester(connListSize)
	connPool, err := tester.setupConnectionPool()
	if err != nil {
		t.Fatalf("failed to setup connection pool: err=%s", err)
	}
	handler := tester.setupHandler(func(msg cleisthenes.Message) {
		if msg.GetRbc().Type != pb.RBC_VAL {
			t.Fatalf(fmt.Sprintf("expected message type is %s, but got %s", string(pb.RBCType_name[int32(pb.RBC_VAL)]), string(msg.GetRbc().Type)))
		}
		payload := &samplePayload{}
		if err := json.Unmarshal(msg.GetRbc().Payload, payload); err != nil {
			t.Fatalf("failed to unmarshal payload: err=%s", err)
		}

		if payload.Data != textList[payload.Id] {
			t.Fatalf(fmt.Sprintf("expected message is %s, but got %s", textList[payload.Id], payload.Data))
		}

		tester.msgRecvChecker.check(payload.Id)

		wg.Done()
	})

	tester.run(connPool, handler)

	msgList := make([]pb.Message, len(textList))
	for i, _ := range msgList {
		payload := &samplePayload{
			Data: textList[i],
			Id:   i,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal sample payload: err=%s", err)
		}
		msgList[i] = pb.Message{
			Payload: &pb.Message_Rbc{
				Rbc: &pb.RBC{
					Payload: data,
					Type:    pb.RBC_VAL,
				},
			},
		}
	}

	connPool.DistributeMessage(msgList)

	wg.Wait()

	return tester.checkResult()
}

func executeShareMessage(t *testing.T, text string, connListSize int) int {
	wg := &sync.WaitGroup{}
	wg.Add(connListSize)

	tester := newConnectionTester(connListSize)
	connPool, err := tester.setupConnectionPool()
	if err != nil {
		t.Fatalf("failed to setup connection pool: err=%s", err)
	}
	handler := tester.setupHandler(func(msg cleisthenes.Message) {
		if !bytes.Equal(msg.GetRbc().Payload, []byte(text)) {
			t.Fatalf("expected message payload is %s, but got %s", msg.GetRbc().Payload, []byte(text))
		}
		tester.msgRecvCounter.count()
		wg.Done()
	})

	tester.run(connPool, handler)

	connPool.ShareMessage(pb.Message{
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload: []byte(text),
				Type:    pb.RBC_VAL,
			},
		},
	})

	wg.Wait()
	return tester.countResult()
}

// Text case for ShareMessage with NodeNum = 3
func TestConnectionPool_ShareMessage(t *testing.T) {
	result := executeShareMessage(t, "jang", 3)
	if result != 3 {
		t.Fatalf("expected result is %d, but got %d", 3, result)
	}
}

// Test case for NodeNum == MessageNum (Both of them are 3) at DistributeMessage
func TestConnectionPool_DistributeMessage_EqualNum(t *testing.T) {
	textList := []string{"jang", "gun", "hee"}
	result := executeDistributeMessage(t, textList, 3)

	trueCount := 0
	for _, check := range result {
		if check {
			trueCount++
		}
	}
	if trueCount != 3 {
		t.Fatalf("expected trueCount is %d, but got %d", 3, trueCount)
	}
}

// Test case for NodeNum < MessageNum (NodeNum is 3, MessageNum is 4) at DistributeMessage
func TestConnectionPool_DistributeMessage_MessageNumBigger(t *testing.T) {
	textList := []string{"jang", "gun", "hee", "kim"}
	result := executeDistributeMessage(t, textList, 3)

	trueCount := 0
	for _, check := range result {
		if check {
			trueCount++
		}
	}
	if trueCount != 3 {
		t.Fatalf("expected trueCount is %d, but got %d", 3, trueCount)
	}
}

// Test case for NodeNum > MessageNum (NodeNum is 3, MessageNum is 2) at DistributeMessage
func TestConnectionPool_DistributeMessage_NodeNumBigger(t *testing.T) {
	textList := []string{"jang", "gun"}
	result := executeDistributeMessage(t, textList, 3)

	trueCount := 0
	for _, check := range result {
		if check {
			trueCount++
		}
	}
	if trueCount != len(textList) {
		t.Fatalf("expected trueCount is %d, but got %d", 3, trueCount)
	}
}

func TestConnectionPool_Add(t *testing.T) {
	mockStreamWrapper := mock.NewStreamWrapper()
	cp := cleisthenes.NewConnectionPool()

	connNum := 5
	connList := make([]cleisthenes.Connection, connNum)
	var err error

	for i, _ := range connList {
		connId := "TestConnection" + strconv.Itoa(i+1)
		addr := cleisthenes.Address{Ip: "127.0.0.1", Port: uint16(8000 + i)}
		connList[i], err = cleisthenes.NewConnection(addr, connId, mockStreamWrapper)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, conn := range connList {
		cp.Add(conn.Ip(), conn)
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
	mockStreamWrapper := mock.NewStreamWrapper()
	cp := cleisthenes.NewConnectionPool()

	connNum := 5
	connList := make([]cleisthenes.Connection, connNum)
	var err error

	for i, _ := range connList {
		connId := "TestConnection" + strconv.Itoa(i+1)
		addr := cleisthenes.Address{Ip: "127.0.0.1", Port: uint16(8000 + i)}
		connList[i], err = cleisthenes.NewConnection(addr, connId, mockStreamWrapper)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, conn := range connList {
		cp.Add(conn.Ip(), conn)
	}

	deleteNum := 2
	deleteConnList := make([]cleisthenes.Address, deleteNum)

	for i, _ := range deleteConnList {
		deleteConnList[i] = connList[i].Ip()
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
