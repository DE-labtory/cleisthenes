package bba

import "github.com/DE-labtory/cleisthenes"

type binarySet struct {
	items map[cleisthenes.Binary]bool
}

func newBinarySet() *binarySet {
	return &binarySet{
		items: make(map[cleisthenes.Binary]bool),
	}
}

func (s *binarySet) exist(bin cleisthenes.Binary) bool {
	_, ok := s.items[bin]
	return ok
}

func (s *binarySet) union(bin cleisthenes.Binary) {
	s.items[bin] = true
}

func (s *binarySet) toList() []cleisthenes.Binary {
	result := make([]cleisthenes.Binary, 0)
	for bin, exist := range s.items {
		if !exist {
			continue
		}
		result = append(result, bin)
	}
	return result
}
