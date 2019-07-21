package rbc

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/DE-labtory/cleisthenes/rbc/merkletree"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/test/mock"

	"github.com/klauspost/reedsolomon"
)

type mockNode struct {
	owner   cleisthenes.Member
	rbcList []*RBC
}

func setUpMockNode(t *testing.T, n int, f int, owner cleisthenes.Member, memberMap *cleisthenes.MemberMap) *mockNode {
	connMemberMap := cleisthenes.NewMemberMap()
	for _, member := range memberMap.Members() {
		if !reflect.DeepEqual(member, owner) {
			connMemberMap.Add(&member)
		}
	}

	bc := setUpMockBC(t, connMemberMap)
	dataChannel := cleisthenes.NewDataChannel(n)
	rbcList := make([]*RBC, 0)
	for _, member := range memberMap.Members() {
		rbc, _ := New(n, f, 1, owner, member, bc, dataChannel)
		rbcList = append(rbcList, rbc)
	}

	return &mockNode{
		owner:   owner,
		rbcList: rbcList,
	}
}

func setUpMockBC(t *testing.T, memberMap *cleisthenes.MemberMap) *mock.Broadcaster {
	bc := &mock.Broadcaster{
		ConnMap:                make(map[cleisthenes.Address]mock.Connection),
		BroadcastedMessageList: make([]pb.Message, 0),
	}

	for _, member := range memberMap.Members() {
		conn := mock.Connection{
			ConnId: member.Address.String(),
		}

		conn.SendFunc = func(msg pb.Message, successCallBack func(interface{}), errCallBack func(error)) {
			bc.BroadcastedMessageList = append(bc.BroadcastedMessageList, msg)
		}

		bc.ConnMap[member.Address] = conn
	}

	return bc
}

func setUpValReqList(t *testing.T, n int, f int, data []byte) []*ValRequest {
	enc, err := reedsolomon.New(n-2*f, 2*f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	shards, err := shard(enc, data)
	if err != nil {
		t.Fatalf("error in shard : %s", err.Error())
	}

	tree, err := merkletree.New(shards)
	if err != nil {
		t.Fatalf("error in make merkle tree : %s", err.Error())
	}

	valReqList := make([]*ValRequest, 0)

	rootHash := tree.MerkleRoot()

	for _, shard := range shards {
		rootPath, indexes, err := tree.MerklePath(shard)
		if err != nil {
			t.Fatalf("error in GetMerklePath : %s", err.Error())
		}

		valReqList = append(valReqList, &ValRequest{
			RootHash: rootHash,
			Data:     shard,
			RootPath: rootPath,
			Indexes:  indexes,
		})
	}

	return valReqList
}

// scenario
// 4 rbc instances
// shard data with 4
// send each node sharded data as VALUE request
// each node handle message
func Test_RBC_handleValueRequest(t *testing.T) {
	n := 4
	f := 1

	data := []byte("this will be sharded")
	valReqList := setUpValReqList(t, n, f, data)

	rbcList := make([]*RBC, 0)
	bcList := make([]*mock.Broadcaster, 0)

	for idx := 0; idx < n; idx++ {
		ownAddr := cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: uint16(8000 + idx),
		}

		memberMap := cleisthenes.NewMemberMap()
		owner := cleisthenes.NewMember(ownAddr.Ip, ownAddr.Port)
		memberMap.Add(owner)
		bc := setUpMockBC(t, memberMap)
		dataChannel := cleisthenes.NewDataChannel(n)

		rbc, _ := New(n, f, 1, *owner, cleisthenes.Member{}, bc, dataChannel)
		rbcList = append(rbcList, rbc)
		bcList = append(bcList, bc)
	}

	for idx, rbc := range rbcList {
		ownAddr := cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: uint16(8000 + idx),
		}
		owner := cleisthenes.Member{
			Address: ownAddr,
		}
		if err := rbc.handleValueRequest(owner, valReqList[idx]); err != nil {
			t.Fatalf("handle val request faild with error : %s", err.Error())
		}

		if !rbc.valReceived.Value() {
			t.Fatalf("rbc instance %s fail to receive val request", rbc.proposer)
		}

		msg := bcList[idx].BroadcastedMessageList[0]

		req, ok := msg.Payload.(*pb.Message_Rbc)
		if !ok {
			t.Fatalf("expected payload type is %+v, but got %+v", pb.Message_Rbc{}, req)
		}

		receivedVal := &ValRequest{}
		if err := json.Unmarshal(req.Rbc.Payload, receivedVal); err != nil {
			t.Fatalf("unmarshal val request failed with error: %s", err.Error())
		}

		if !merkletree.ValidatePath(receivedVal.Data, receivedVal.RootHash, receivedVal.RootPath, receivedVal.Indexes) {
			t.Fatalf("val request has invalid datas")
		}
	}
}

// scenario
// 4 node(n0, n1, n2, n3), 1 byzantine.
// n0 sends VALUE message to n0, n1, n2, n3.
// each nodes sends ECHO messages.
func Test_RBC_handleEchoRequest(t *testing.T) {
	n := 4
	f := 1

	memberMap := cleisthenes.NewMemberMap()
	for idx := 0; idx < n; idx++ {
		member := cleisthenes.NewMember("127.0.0.1", uint16(8000+idx))
		memberMap.Add(member)
	}

	nodeList := make([]*mockNode, 0)
	for _, member := range memberMap.Members() {
		node := setUpMockNode(t, n, f, member, memberMap)
		nodeList = append(nodeList, node)
	}

	valReqList := setUpValReqList(t, n, f, []byte("data"))

	for _, node := range nodeList {
		for i, valReq := range valReqList {
			node.rbcList[0].handleEchoRequest(nodeList[i].owner, &EchoRequest{*valReq})
		}
	}

	for _, node := range nodeList {
		if node.rbcList[0].countEchos(valReqList[0].RootHash) != 4 {
			t.Fatalf("error - expected : %d, got : %d", 4, node.rbcList[0].countEchos(valReqList[0].RootHash))
		}
	}
}

type Data struct {
	Field1 string
	Field2 int
}

func Test_RBC_interpolate_primitive(t *testing.T) {
	n := 4
	f := 1

	data := []byte("data block")
	valReqs := setUpValReqList(t, 4, 1, data)
	echoReqRepo := NewEchoReqRepository()

	for idx, req := range valReqs {
		addr := cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: uint16(8000 + idx),
		}
		echoReqRepo.Save(addr, &EchoRequest{*req})
	}

	enc, err := reedsolomon.New(n-2*f, 2*f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	dataChannel := cleisthenes.NewDataChannel(n)

	rbc := &RBC{
		n:               n,
		f:               f,
		numDataShards:   n - 2*f,
		numParityShards: 2 * f,
		output:          &output{sync.RWMutex{}, nil},
		contentLength:   &contentLength{sync.RWMutex{}, uint64(len(data))},
		enc:             enc,
		echoReqRepo:     echoReqRepo,
		dataSender:      dataChannel,
	}

	rootHash := valReqs[0].RootHash

	value, err := rbc.tryDecodeValue(rootHash)
	if err != nil {
		t.Fatalf("error in interpolate data : %s", err.Error())
	}

	if !bytes.Equal(data, value) {
		t.Fatalf("not equal original data and interpolated data - expected : %s, got : %s", data, rbc.output.value())
	}
}

func Test_RBC_interpolate_struct(t *testing.T) {
	n := 4
	f := 1

	data := Data{
		Field1: "kim",
		Field2: 26,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("error in josn marshal : %s", err.Error())
	}

	valReqs := setUpValReqList(t, 4, 1, payload)
	echoReqRepo := NewEchoReqRepository()

	for idx, req := range valReqs {
		addr := cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: uint16(8000 + idx),
		}
		echoReqRepo.Save(addr, &EchoRequest{*req})
	}

	enc, err := reedsolomon.New(n-2*f, 2*f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	dataChannel := cleisthenes.NewDataChannel(n)

	rbc := &RBC{
		n:               n,
		f:               f,
		numDataShards:   n - 2*f,
		numParityShards: 2 * f,
		output:          &output{sync.RWMutex{}, nil},
		contentLength:   &contentLength{sync.RWMutex{}, uint64(len(payload))},
		enc:             enc,
		echoReqRepo:     echoReqRepo,
		dataSender:      dataChannel,
	}

	rootHash := valReqs[0].RootHash

	value, err := rbc.tryDecodeValue(rootHash)
	if err != nil {
		t.Fatalf("error in interpolate data : %s", err.Error())
	}

	if !bytes.Equal(payload, value) {
		t.Fatalf("not equal original data and interpolated data (bytes) - expected : %x, got : %x", data, rbc.output.value())
	}

	interpolatedData := &Data{}
	if err := json.Unmarshal(payload, interpolatedData); err != nil {
		t.Fatalf("error in json unmarshal : %s", err.Error())
	}

	if !reflect.DeepEqual(*interpolatedData, data) {
		t.Fatalf("not equal original data and interpolated data (struct)  - expected : %+v, got : %+v", data, interpolatedData)
	}
}

func Test_MakeRequest(t *testing.T) {
	n := 10
	f := 3

	data := []byte("data")
	enc, err := reedsolomon.New(n-2*f, 2*f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	shards, err := shard(enc, data)
	if err != nil {
		t.Fatalf("error in shard : %s", err.Error())
	}

	reqs, err := makeRequest(shards)
	if err != nil {
		t.Fatalf("error in makeRequest : %s", err.Error())
	}

	if len(reqs) != n {
		t.Fatalf("not equal request size - expected=%d, got=%d", n, len(reqs))
	}

}

func Test_shard(t *testing.T) {
	n := 10
	f := 3

	data := []byte("data")
	enc, err := reedsolomon.New(n-2*f, 2*f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	shards, err := shard(enc, data)
	if err != nil {
		t.Fatalf("error in shard : %s", err.Error())
	}

	if len(shards) != n {
		t.Fatalf("not equal shards size - expected=%d, got=%d", n, len(shards))
	}

	var value []byte
	for _, data := range shards[:n-2*f] {
		value = append(value, data.Bytes()...)
	}

	if !bytes.Equal(data, value[:len(data)]) {
		t.Fatalf("not equal data and interpolated output - expected=%s, got=%s", data, value[:len(data)])
	}
}
