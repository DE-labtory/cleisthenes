package acs

import (
	"reflect"
	"testing"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/rbc"
)

func TestRBCRepository_Save(t *testing.T) {
	mem := cleisthenes.Member{
		cleisthenes.Address{
			Ip:   "localhost",
			Port: 8000,
		},
	}

	rbcRepo := NewRBCRepository()
	r := rbc.RBC{}
	rbcRepo.Save(mem, &r)

	_, ok := rbcRepo.rbcMap[mem]
	if !ok {
		t.Fatalf("rbc %v is not saved", rbcRepo.rbcMap[mem])
	}
}

func TestRBCRepository_Find(t *testing.T) {
	mem := cleisthenes.Member{
		cleisthenes.Address{
			Ip:   "localhost",
			Port: 8000,
		},
	}

	rbcRepo := NewRBCRepository()
	r := rbc.RBC{}
	rbcRepo.Save(mem, &r)

	_, err := rbcRepo.Find(mem)
	if err != nil {
		t.Fatalf("rbc %v is not found, err : %s", rbcRepo.rbcMap[mem], err.Error())
	}
}

func TestRBCRepository_FindAll(t *testing.T) {
	mem := cleisthenes.Member{
		cleisthenes.Address{
			Ip:   "localhost",
			Port: 8000,
		},
	}
	mem2 := cleisthenes.Member{
		cleisthenes.Address{
			Ip:   "localhost",
			Port: 8001,
		},
	}

	rbcRepo := NewRBCRepository()
	r := rbc.RBC{}
	rbcRepo.Save(mem, &r)
	rbcRepo.Save(mem2, &r)

	if len(rbcRepo.FindAll()) != 2 {
		t.Fatalf("invalid repository size")
	}

	for _, comp := range rbcRepo.FindAll() {
		rbc, _ := comp.(*rbc.RBC)
		if !reflect.DeepEqual(rbc, &r) {
			t.Fatalf("invalid repository value")
		}
	}
}
