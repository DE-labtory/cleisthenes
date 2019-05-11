package bba

type binarySet struct {
	items map[Binary]bool
}

func newBinarySet() *binarySet {
	return &binarySet{
		items: make(map[Binary]bool),
	}
}

func (s *binarySet) exist(bin Binary) bool {
	_, ok := s.items[bin]
	return ok
}

func (s *binarySet) union(bin Binary) {
	s.items[bin] = true
}
