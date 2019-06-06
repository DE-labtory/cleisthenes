package mock

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type Connection struct {
	ConnId   cleisthenes.ConnId
	SendFunc func(msg pb.Message, successCallBack func(interface{}), errCallBack func(error))
}

func (c *Connection) Send(msg pb.Message, successCallBack func(interface{}), errCallBack func(error)) {
	c.SendFunc(msg, successCallBack, errCallBack)
}
func (c *Connection) Ip() cleisthenes.Address {
	return cleisthenes.Address{}
}
func (c *Connection) Id() cleisthenes.ConnId {
	return c.ConnId
}
func (c *Connection) Close() {}
func (c *Connection) Start() error {
	return nil
}
func (c *Connection) Handle(handler cleisthenes.Handler) {}

type Broadcaster struct {
	ConnMap                map[cleisthenes.Address]Connection
	BroadcastedMessageList []pb.Message
}

func (b *Broadcaster) ShareMessage(msg pb.Message) {
	for _, conn := range b.ConnMap {
		conn.Send(msg, nil, nil)
	}
}

func (b *Broadcaster) DistributeMessage(msgList []pb.Message) {}
