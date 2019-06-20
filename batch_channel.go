package cleisthenes

type BatchMessage struct {
	Batch map[Member][]byte
}

type BatchSender interface {
	Send(msg BatchMessage)
}

type BatchReceiver interface {
	Receive() <-chan BatchMessage
}

type BatchChannel struct {
	buffer chan BatchMessage
}

func NewBatchChannel() *BatchChannel {
	return &BatchChannel{
		buffer: make(chan BatchMessage, 1),
	}
}

func (c *BatchChannel) Send(msg BatchMessage) {
	c.buffer <- msg
}

func (c *BatchChannel) Receive() <-chan BatchMessage {
	return c.buffer
}
