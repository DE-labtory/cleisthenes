package acs

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
)

type BBA interface {
	HandleInput(bvalRequest *bba.BvalRequest) error
	HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error
	Close()
	Accept() bool
}

type BBARepository struct {
	lock   sync.RWMutex
	bbaMap map[cleisthenes.Member]BBA
}

func NewBBARepository() *BBARepository {
	return &BBARepository{
		lock:   sync.RWMutex{},
		bbaMap: make(map[cleisthenes.Member]BBA),
	}
}

func (r *BBARepository) Save(mem cleisthenes.Member, bba BBA) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.bbaMap[mem] = bba
}

func (r *BBARepository) Find(mem cleisthenes.Member) (BBA, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.bbaMap[mem]
	if !ok {
		return nil, ErrNoMemberMatchingRequest
	}
	return r.bbaMap[mem], nil
}

func (r *BBARepository) FindAll() []BBA {
	r.lock.Lock()
	defer r.lock.Unlock()

	bbaList := make([]BBA, 0)
	for _, bba := range r.bbaMap {
		bbaList = append(bbaList, bba)
	}
	return bbaList
}
