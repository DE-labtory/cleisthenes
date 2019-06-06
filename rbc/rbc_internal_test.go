package rbc

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/DE-labtory/cleisthenes/rbc/merkletree"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/test/mock"

	"github.com/klauspost/reedsolomon"
)

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
	enc, err := reedsolomon.New(n-f, f)
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
	var n int = 4
	var f int = 1
	config := cleisthenes.Config{
		N: n,
		F: f,
	}

	data := []byte("this will be sharded")
	valReqList := setUpValReqList(t, n, f, data)

	rbcList := make([]*RBC, 0)
	bcList := make([]*mock.Broadcaster, 0)

	for idx := 0; idx < n; idx++ {
		ownAddr := cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: uint16(8000 + idx),
		}
		config.Address = ownAddr

		memberMap := cleisthenes.NewMemberMap()
		memberMap.Add(cleisthenes.NewMember(ownAddr.Ip, ownAddr.Port))
		bc := setUpMockBC(t, memberMap)

		rbc := New(config, cleisthenes.Member{}, nil)
		rbc.broadcaster = bc
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

		if !rbc.valReceived {
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

func Test_RBC_interpolate(t *testing.T) {
	var n int = 4
	var f int = 1

	data := []byte("data block")
	valReqs := setUpValReqList(t, 4, 1, data)
	echoReqRepo, _ := NewEchoReqRepository()

	for idx, req := range valReqs {
		addr := cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: uint16(8000 + idx),
		}
		echoReqRepo.Save(addr, &EchoRequest{*req})
	}

	enc, err := reedsolomon.New(n-f, f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	rbc := &RBC{
		n:               n,
		f:               f,
		numDataShards:   n - f,
		numParityShards: f,
		contentLength:   uint64(len(data)),
		enc:             enc,
		echoReqRepo:     echoReqRepo,
	}

	rootHash := valReqs[0].RootHash

	value, err := rbc.interpolate(rootHash)
	if err != nil {
		t.Fatalf("error in interpolate data : %s", err.Error())
	}

	if !bytes.Equal(data, value) {
		t.Fatalf("not equal original data and interpolated data - expected : %s, got : %s", data, value)
	}
}

func Test_MakeRequest(t *testing.T) {
	var N int = 10
	var F int = 3

	data := []byte("data")
	enc, err := reedsolomon.New(N-F, F)
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

	if len(reqs) != N {
		t.Fatalf("not equal request size - expected=%d, got=%d", N, len(reqs))
	}

}

func Test_shard(t *testing.T) {
	var N int = 10
	var F int = 3

	data := []byte("data")
	enc, err := reedsolomon.New(N-F, F)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	shards, err := shard(enc, data)
	if err != nil {
		t.Fatalf("error in shard : %s", err.Error())
	}

	if len(shards) != N {
		t.Fatalf("not equal shards size - expected=%d, got=%d", N, len(shards))
	}

	var value []byte
	for _, data := range shards[:N-F] {
		value = append(value, data.Bytes()...)
	}

	if !bytes.Equal(data, value[:len(data)]) {
		t.Fatalf("not equal data and interpolated output - expected=%s, got=%s", data, value[:len(data)])
	}
}
