package mock

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type StreamWrapper struct {
	InternalChan chan *pb.Message
	CloseChan    chan struct{}
}

func NewStreamWrapper() *StreamWrapper {
	return &StreamWrapper{
		InternalChan: make(chan *pb.Message),
		CloseChan:    make(chan struct{}, 1),
	}
}

func (c *StreamWrapper) Send(msg *pb.Message) error {
	c.InternalChan <- msg
	return nil
}

func (c *StreamWrapper) Recv() (*pb.Message, error) {
	select {
	case m := <-c.InternalChan:
		return m, nil
	}
}

func (c *StreamWrapper) Close() {
	c.CloseChan <- struct{}{}
}

func (c *StreamWrapper) GetStream() cleisthenes.Stream {
	return nil
}
