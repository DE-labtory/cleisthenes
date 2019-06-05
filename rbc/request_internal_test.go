package rbc

import (
	"reflect"
	"testing"

	"github.com/DE-labtory/cleisthenes"
)

func TestValReqRepository_Find(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}

	valReqList, _ := NewValReqRepository()
	val := ValRequest{nil, nil, nil}
	valReqList.Save(addr, &val)
	_, ok := valReqList.Find(addr)
	if ok != nil {
		t.Fatalf("request %v is not found.", valReqList.recv[addr])
	}
}
func TestValReqRepository_FindAll(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}
	addr2 := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8001,
	}

	valReqList, _ := NewValReqRepository()
	val := ValRequest{nil, nil, nil}
	valReqList.Save(addr, &val)
	valReqList.Save(addr2, &val)

	if len(valReqList.FindAll()) != 2 {
		t.Fatalf("request %v,%v is not found all.", valReqList.recv[addr], valReqList.recv[addr2])
	}
	for _, request := range valReqList.FindAll() {
		valReq, _ := request.(*ValRequest)
		if !reflect.DeepEqual(valReq, &val) {
			t.Fatalf("request %v,%v is not found all.", valReqList.recv[addr], valReqList.recv[addr2])
		}
	}
}
func TestValReqRepository_Save(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}

	valReqList, _ := NewValReqRepository()
	val := ValRequest{nil, nil, nil}
	valReqList.Save(addr, &val)
	_, ok := valReqList.recv[addr]
	if !ok {
		t.Fatalf("request %v is not saved", valReqList.recv[addr])
	}
}
func TestValReqRepository_NewValReqRepository(t *testing.T) {
	_, err := NewValReqRepository()
	if err != nil {
		t.Fatalf("val request repository is not created.")
	}
}
func TestNewEchoReqRepository(t *testing.T) {
	_, err := NewEchoReqRepository()
	if err != nil {
		t.Fatalf("echo requst repository is not created.")
	}
}
func TestEchoReqRepository_Find(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}

	echoReqList, _ := NewEchoReqRepository()
	val := ValRequest{nil, nil, nil}
	echo := EchoRequest{val}
	echoReqList.Save(addr, &echo)
	_, ok := echoReqList.Find(addr)
	if ok != nil {
		t.Fatalf("request %v is not found.", echoReqList.recv[addr])
	}

}
func TestEchoReqRepository_FindAll(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}
	addr2 := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8001,
	}

	echoReqList, _ := NewEchoReqRepository()
	val := ValRequest{nil, nil, nil}
	echo := EchoRequest{val}
	echoReqList.Save(addr, &echo)
	echoReqList.Save(addr2, &echo)
	if len(echoReqList.FindAll()) != 2 {
		t.Fatalf("request %v,%v is not found all.", echoReqList.recv[addr], echoReqList.recv[addr2])
	}
	for _, request := range echoReqList.FindAll() {
		echoReq, _ := request.(*EchoRequest)
		if !reflect.DeepEqual(echoReq, &echo) {
			t.Fatalf("request %v,%v is not found all.", echoReqList.recv[addr], echoReqList.recv[addr2])
		}
	}
}
func TestEchoReqRepository_Save(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}

	echoReqList, _ := NewEchoReqRepository()
	val := ValRequest{nil, nil, nil}
	echo := EchoRequest{val}
	echoReqList.Save(addr, &echo)
	_, ok := echoReqList.recv[addr]
	if !ok {
		t.Fatalf("request %v is not saved", echoReqList.recv[addr])
	}
}
func TestNewReadyReqRepository(t *testing.T) {
	_, err := NewReadyReqRepository()
	if err != nil {
		t.Fatalf("ready request repository is not created.")
	}

}
func TestReadyReqRepository_FindAll(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}
	addr2 := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8001,
	}

	readyReqList, _ := NewReadyReqRepository()
	ready := ReadyRequest{nil}
	readyReqList.Save(addr, &ready)
	readyReqList.Save(addr2, &ready)
	if len(readyReqList.FindAll()) != 2 {
		t.Fatalf("request %v,%v is not found all.", readyReqList.recv[addr], readyReqList.recv[addr2])
	}
	for _, request := range readyReqList.FindAll() {
		readyReq, _ := request.(*ReadyRequest)
		if !reflect.DeepEqual(readyReq, &ready) {
			t.Fatalf("request %v,%v is not found all.", readyReqList.recv[addr], readyReqList.recv[addr2])
		}
	}

}
func TestReadyReqRepository_Save(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}

	readyReqList, _ := NewReadyReqRepository()
	ready := ReadyRequest{nil}
	readyReqList.Save(addr, &ready)
	_, ok := readyReqList.recv[addr]
	if !ok {
		t.Fatalf("request %v is not saved", readyReqList.recv[addr])
	}

}
func TestReadyReqRepository_Find(t *testing.T) {
	addr := cleisthenes.Address{
		Ip:   "localhost",
		Port: 8000,
	}

	readyReqList, _ := NewReadyReqRepository()
	ready := ReadyRequest{nil}
	readyReqList.Save(addr, &ready)
	_, ok := readyReqList.Find(addr)
	if ok != nil {
		t.Fatalf("request %v is not found.", readyReqList.recv[addr])
	}

}
