package rbc

import (
	"bytes"
	"testing"

	"github.com/DE-labtory/cleisthenes/rbc/merkletree"

	"github.com/DE-labtory/cleisthenes"
	"github.com/klauspost/reedsolomon"
)

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

	enc, err := reedsolomon.New(n-2*f, 2*f)
	if err != nil {
		t.Fatalf("error in reedsolomon : %s", err.Error())
	}

	rbc := &RBC{
		n:               n,
		f:               f,
		numDataShards:   n - 2*f,
		numParityShards: 2 * f,
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
	enc, err := reedsolomon.New(N-2*F, 2*F)
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
