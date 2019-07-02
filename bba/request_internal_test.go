package bba

import (
	"testing"

	"github.com/DE-labtory/cleisthenes"
)

func TestDefaultIncomingReqRepository(t *testing.T) {
	repo := newDefaultIncomingRequestRepository()
	addr1, _ := cleisthenes.ToAddress("localhost:8080")
	addr2, _ := cleisthenes.ToAddress("localhost:8081")
	addr3, _ := cleisthenes.ToAddress("localhost:8082")

	repo.Save(1, addr1, &BvalRequest{Value: cleisthenes.One})
	repo.Save(1, addr2, &BvalRequest{Value: cleisthenes.Zero})

	result := repo.Find(1)
	if len(result) != 2 {
		t.Fatalf("expected length of result is %d, but got %d", 2, len(result))
	}

	repo.Save(2, addr3, &AuxRequest{Value: cleisthenes.One})
	result = repo.Find(1)
	if len(result) != 2 {
		t.Fatalf("expected length of result is %d, but got %d", 2, len(result))
	}

	result = repo.Find(2)
	if len(result) != 1 {
		t.Fatalf("expected length of result is %d, but got %d", 2, len(result))
	}
}
