package cleisthenes

type Binary = bool

const (
	One  Binary = true
	Zero        = false
)

type BinaryState struct {
	value     Binary
	undefined bool
}

func (b *BinaryState) Set(bin Binary) {
	b.value = bin
	b.undefined = false
}

func (b *BinaryState) Value() Binary {
	return b.value
}

func (b *BinaryState) Undefined() bool {
	return b.undefined
}

func NewBinaryState() *BinaryState {
	return &BinaryState{
		undefined: true,
	}
}
