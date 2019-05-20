package rbc

import (
	"reflect"
	"testing"
)

func TestValReqRepository_Find(t *testing.T) {
	valReqList, _ := NewValReqRepository()
	val := ValRequest{nil, nil, nil}
	valReqList.Save("ID1", &val)
	_, ok := valReqList.Find("ID1")
	if ok != nil {
		t.Fatalf("request %v is not found.", valReqList.recv["ID1"])
	}
}
func TestValReqRepository_FindAll(t *testing.T) {
	valReqList, _ := NewValReqRepository()
	val := ValRequest{nil, nil, nil}
	valReqList.Save("ID1", &val)
	valReqList.Save("ID2", &val)

	if len(valReqList.FindAll()) != 2 {
		t.Fatalf("request %v,%v is not found all.", valReqList.recv["ID1"], valReqList.recv["ID2"])
	}
	for _, request := range valReqList.FindAll() {
		valReq, _ := request.(*ValRequest)
		if !reflect.DeepEqual(valReq, &val) {
			t.Fatalf("request %v,%v is not found all.", valReqList.recv["ID1"], valReqList.recv["ID2"])
		}
	}
}
func TestValReqRepository_Save(t *testing.T) {
	valReqList, _ := NewValReqRepository()
	val := ValRequest{nil, nil, nil}
	valReqList.Save("ID1", &val)
	_, ok := valReqList.recv["ID1"]
	if !ok {
		t.Fatalf("request %v is not saved", valReqList.recv["ID1"])
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
	echoReqList, _ := NewEchoReqRepository()
	val := ValRequest{nil, nil, nil}
	echo := EchoRequest{val}
	echoReqList.Save("ID1", &echo)
	_, ok := echoReqList.Find("ID1")
	if ok != nil {
		t.Fatalf("request %v is not found.", echoReqList.recv["ID1"])
	}

}
func TestEchoReqRepository_FindAll(t *testing.T) {
	echoReqList, _ := NewEchoReqRepository()
	val := ValRequest{nil, nil, nil}
	echo := EchoRequest{val}
	echoReqList.Save("ID1", &echo)
	echoReqList.Save("ID2", &echo)
	if len(echoReqList.FindAll()) != 2 {
		t.Fatalf("request %v,%v is not found all.", echoReqList.recv["ID1"], echoReqList.recv["ID2"])
	}
	for _, request := range echoReqList.FindAll() {
		echoReq, _ := request.(*EchoRequest)
		if !reflect.DeepEqual(echoReq, &echo) {
			t.Fatalf("request %v,%v is not found all.", echoReqList.recv["ID1"], echoReqList.recv["ID2"])
		}
	}
}
func TestEchoReqRepository_Save(t *testing.T) {
	echoReqList, _ := NewEchoReqRepository()
	val := ValRequest{nil, nil, nil}
	echo := EchoRequest{val}
	echoReqList.Save("ID1", &echo)
	_, ok := echoReqList.recv["ID1"]
	if !ok {
		t.Fatalf("request %v is not saved", echoReqList.recv["ID1"])
	}
}
func TestNewReadyReqRepository(t *testing.T) {
	_, err := NewReadyReqRepository()
	if err != nil {
		t.Fatalf("ready request repository is not created.")
	}

}
func TestReadyReqRepository_FindAll(t *testing.T) {
	readyReqList, _ := NewReadyReqRepository()
	ready := ReadyRequest{nil}
	readyReqList.Save("ID1", &ready)
	readyReqList.Save("ID2", &ready)
	if len(readyReqList.FindAll()) != 2 {
		t.Fatalf("request %v,%v is not found all.", readyReqList.recv["ID1"], readyReqList.recv["ID2"])
	}
	for _, request := range readyReqList.FindAll() {
		readyReq, _ := request.(*ReadyRequest)
		if !reflect.DeepEqual(readyReq, &ready) {
			t.Fatalf("request %v,%v is not found all.", readyReqList.recv["ID1"], readyReqList.recv["ID2"])
		}
	}

}
func TestReadyReqRepository_Save(t *testing.T) {
	readyReqList, _ := NewReadyReqRepository()
	ready := ReadyRequest{nil}
	readyReqList.Save("ID1", &ready)
	_, ok := readyReqList.recv["ID1"]
	if !ok {
		t.Fatalf("request %v is not saved", readyReqList.recv["ID1"])
	}

}
func TestReadyReqRepository_Find(t *testing.T) {
	readyReqList, _ := NewReadyReqRepository()
	ready := ReadyRequest{nil}
	readyReqList.Save("ID1", &ready)
	_, ok := readyReqList.Find("ID1")
	if ok != nil {
		t.Fatalf("request %v is not found.", readyReqList.recv["ID1"])
	}

}
