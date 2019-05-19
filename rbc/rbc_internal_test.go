package rbc

import (
	"testing"

	"github.com/klauspost/reedsolomon"
)

func Test_RBC_broadcast(t *testing.T) {

}

func Test_RBC_handleValueRequest(t *testing.T) {

}

func Test_RBC_handleEchoRequest(t *testing.T) {

}

func Test_RBC_handleReadyRequest(t *testing.T) {

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

func Test_interpolate(t *testing.T) {

}

func Test_validateMessage(t *testing.T) {

}

func Test_shard(t *testing.T) {
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

	if len(shards) != N {
		t.Fatalf("not equal shards size - expected=%d, got=%d", N, len(shards))
	}
}
