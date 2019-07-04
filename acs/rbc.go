package acs

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type RBC interface {
	HandleInput(data []byte) error
	HandleMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error
	Close()
}

type RBCRepository struct {
	lock   sync.RWMutex
	rbcMap map[cleisthenes.Member]RBC
}

func NewRBCRepository() *RBCRepository {
	return &RBCRepository{
		lock:   sync.RWMutex{},
		rbcMap: make(map[cleisthenes.Member]RBC),
	}
}

func (r *RBCRepository) Save(mem cleisthenes.Member, rbc RBC) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.rbcMap[mem] = rbc
}

func (r *RBCRepository) Find(mem cleisthenes.Member) (RBC, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.rbcMap[mem]
	if !ok {
		return nil, ErrNoMemberMatchingRequest
	}
	return r.rbcMap[mem], nil
}

func (r *RBCRepository) FindAll() []RBC {
	r.lock.Lock()
	defer r.lock.Unlock()

	rbcList := make([]RBC, 0)
	for _, rbc := range r.rbcMap {
		rbcList = append(rbcList, rbc)
	}
	return rbcList
}
