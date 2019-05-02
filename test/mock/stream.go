package mock

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type MockStreamWrapper struct {
	InternalChan chan *pb.Message
	CloseChan    chan struct{}
}

func NewMockStreamWrapper() *MockStreamWrapper {
	return &MockStreamWrapper{
		InternalChan: make(chan *pb.Message),
		CloseChan:    make(chan struct{}, 1),
	}
}

func (c *MockStreamWrapper) Send(msg *pb.Message) error {
	c.InternalChan <- msg
	return nil
}

func (c *MockStreamWrapper) Recv() (*pb.Message, error) {
	select {
	case m := <-c.InternalChan:
		return m, nil
	}
}

func (c *MockStreamWrapper) Close() {
	c.CloseChan <- struct{}{}
}

func (c *MockStreamWrapper) GetStream() cleisthenes.Stream {
	return nil
}
