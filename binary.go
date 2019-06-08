package cleisthenes

import "sync"

type Binary = bool

const (
	One  Binary = true
	Zero        = false
)

type BinaryState struct {
	sync.RWMutex
	value     Binary
	undefined bool
}

func (b *BinaryState) Set(bin Binary) {
	b.Lock()
	defer b.Unlock()

	b.value = bin
	b.undefined = false
}

func (b *BinaryState) Value() Binary {
	b.Lock()
	defer b.Unlock()

	return b.value
}

func (b *BinaryState) Undefined() bool {
	b.Lock()
	defer b.Unlock()

	return b.undefined
}

func NewBinaryState() *BinaryState {
	return &BinaryState{
		RWMutex:   sync.RWMutex{},
		undefined: true,
	}
}
