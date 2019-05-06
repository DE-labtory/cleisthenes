package bba

type binarySet struct {
	items map[Binary]bool
}

func (s *binarySet) exist(bin Binary) bool {
	return false
}

func (s *binarySet) union(bin Binary) {}
