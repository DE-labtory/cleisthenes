package cleisthenes

type DataMessage struct {
	Member Member
	Data   []byte
}

type DataSender interface {
	Send(msg DataMessage)
}

type DataReceiver interface {
	Receive() <-chan DataMessage
}

type DataChannel struct {
	buffer chan DataMessage
}

func NewDataChannel(size int) *DataChannel {
	return &DataChannel{
		buffer: make(chan DataMessage, size),
	}
}

func (c *DataChannel) Send(msg DataMessage) {
	c.buffer <- msg
}

func (c *DataChannel) Receive() <-chan DataMessage {
	return c.buffer
}
