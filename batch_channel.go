package cleisthenes

// Batch is result of ACS set of contributions of
// at least n-f number of nodes.
// BatchMessage is used between ACS component and Honeybadger component.
//
// After ACS done its own task for its epoch send BatchMessage to
// Honeybadger, then Honeybadger decrypt batch message.
type BatchMessage struct {
	Epoch Epoch
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

func NewBatchChannel(size int) *BatchChannel {
	return &BatchChannel{
		buffer: make(chan BatchMessage, size),
	}
}

func (c *BatchChannel) Send(msg BatchMessage) {
	c.buffer <- msg
}

func (c *BatchChannel) Receive() <-chan BatchMessage {
	return c.buffer
}

// ResultMessage is result of Honeybadger. When Honeybadger receive
// BatchMessage from ACS, it decrypt BatchMessage and use it to
// ResultMessage.Batch field.
//
// Honeybadger knows what epoch of ACS done its task. and Honeybadger
// use that information of epoch and decrypted batch to create ResultMessage
// then send it back to application
type ResultMessage struct {
	Epoch Epoch
	Batch []AbstractTx
}

type ResultSender interface {
	Send(msg ResultMessage)
}

type ResultReceiver interface {
	Receive() <-chan ResultMessage
}

type ResultChannel struct {
	buffer chan ResultMessage
}

func NewResultChannel(size int) *ResultChannel {
	return &ResultChannel{
		buffer: make(chan ResultMessage, size),
	}
}

func (c *ResultChannel) Send(msg ResultMessage) {
	c.buffer <- msg
}

func (c *ResultChannel) Receive() <-chan ResultMessage {
	return c.buffer
}
