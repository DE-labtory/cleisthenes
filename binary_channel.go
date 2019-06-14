package cleisthenes

type BinaryMessage struct {
	Member Member
	Binary Binary
}

type BinarySender interface {
	Send(msg BinaryMessage)
}

type BinaryReceiver interface {
	Receive() <-chan BinaryMessage
}

type BinaryChannel struct {
	buffer chan BinaryMessage
}

func NewBinaryChannel(size int) *BinaryChannel {
	return &BinaryChannel{
		buffer: make(chan BinaryMessage, size),
	}
}

func (c *BinaryChannel) Send(msg BinaryMessage) {
	c.buffer <- msg
}

func (c *BinaryChannel) Receive() <-chan BinaryMessage {
	return c.buffer
}
