package acs

import (
	"reflect"
	"testing"

	"github.com/DE-labtory/cleisthenes/bba"

	"github.com/DE-labtory/cleisthenes"
)

func TestBBARepository_Save(t *testing.T) {
	mem := cleisthenes.Member{
		cleisthenes.Address{
			Ip:   "localhost",
			Port: 8000,
		},
	}

	bbaRepo := NewBBARepository()
	b := bba.BBA{}
	bbaRepo.Save(mem, &b)

	_, ok := bbaRepo.bbaMap[mem]
	if !ok {
		t.Fatalf("rbc %v is not saved", bbaRepo.bbaMap[mem])
	}
}

func TestBBARepository_Find(t *testing.T) {
	mem := cleisthenes.Member{
		cleisthenes.Address{
			Ip:   "localhost",
			Port: 8000,
		},
	}

	bbaRepo := NewBBARepository()
	b := bba.BBA{}
	bbaRepo.Save(mem, &b)

	_, err := bbaRepo.Find(mem)
	if err != nil {
		t.Fatalf("rbc %v is not found, err : %s", bbaRepo.bbaMap[mem], err.Error())
	}
}

func TestBBARepository_FindAll(t *testing.T) {
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

	bbaRepo := NewBBARepository()
	b := bba.BBA{}
	bbaRepo.Save(mem, &b)
	bbaRepo.Save(mem2, &b)

	if len(bbaRepo.FindAll()) != 2 {
		t.Fatalf("invalid repository size")
	}

	for _, comp := range bbaRepo.FindAll() {
		bba, _ := comp.(*bba.BBA)
		if !reflect.DeepEqual(bba, &b) {
			t.Fatalf("invalid repository value")
		}
	}
}
