package bba

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

type binarySet struct {
	lock  sync.RWMutex
	items map[cleisthenes.Binary]bool
}

func newBinarySet() *binarySet {
	return &binarySet{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Binary]bool),
	}
}

func (s *binarySet) exist(bin cleisthenes.Binary) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.items[bin]
	return ok
}

func (s *binarySet) union(bin cleisthenes.Binary) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items[bin] = true
}

func (s *binarySet) toList() []cleisthenes.Binary {
	s.lock.Lock()
	defer s.lock.Unlock()

	result := make([]cleisthenes.Binary, 0)
	for bin, exist := range s.items {
		if !exist {
			continue
		}
		result = append(result, bin)
	}
	return result
}
