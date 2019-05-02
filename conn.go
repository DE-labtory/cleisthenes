package cleisthenes

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnID = string

// message used in HBBFT
type innerMessage struct {
	Message   *pb.Message
	OnErr     func(error)
	OnSuccess func(interface{})
}

// message used with other nodes
type Message struct {
	*pb.Message
	Conn Connection
}

// request handler
type Handler interface {
	ServeRequest(msg Message)
}

type Connection interface {
	Send(msg pb.Message, successCallBack func(interface{}), errCallBack func(error))
	Ip() Address
	Id() ConnID
	Close()
	Start() error
	Handle(handler Handler)
}

type GrpcConnection struct {
	id            ConnID
	ip            Address
	streamWrapper StreamWrapper
	stopFlag      int32
	handler       Handler
	outChan       chan *innerMessage
	readChan      chan *pb.Message
	stopChan      chan struct{}
	sync.RWMutex
}

func NewConnection(ip Address, id ConnID, streamWrapper StreamWrapper) (Connection, error) {
	if streamWrapper == nil {
		return nil, errors.New("fail to create connection ! : streamWrapper is nil")
	}

	return &GrpcConnection{
		id:            id,
		ip:            ip,
		streamWrapper: streamWrapper,
		outChan:       make(chan *innerMessage, 200),
		readChan:      make(chan *pb.Message, 200),
		stopChan:      make(chan struct{}, 1),
	}, nil
}

func (conn *GrpcConnection) Send(msg pb.Message, successCallBack func(interface{}), errCallBack func(error)) {
	conn.Lock()
	defer conn.Unlock()

	m := &innerMessage{
		Message:   &msg,
		OnErr:     errCallBack,
		OnSuccess: successCallBack,
	}

	conn.outChan <- m
}

func (conn *GrpcConnection) Ip() Address {
	return conn.ip
}

func (conn *GrpcConnection) Id() ConnID {
	return conn.id
}

func (conn *GrpcConnection) Close() {
	if conn.isDie() {
		return
	}

	isFisrt := atomic.CompareAndSwapInt32(&conn.stopFlag, int32(0), int32(1))

	if !isFisrt {
		return
	}

	conn.stopChan <- struct{}{}
	conn.Lock()

	conn.streamWrapper.Close()

	conn.Unlock()
}

func (conn *GrpcConnection) Start() error {
	errChan := make(chan error, 1)

	go conn.readStream(errChan)
	go conn.writeStream()

	for !conn.isDie() {
		select {
		case stop := <-conn.stopChan:
			conn.stopChan <- stop
			return nil
		case err := <-errChan:
			return err
		case message := <-conn.readChan:
			if conn.verify(message) {
				if conn.handler != nil {
					m := Message{Message: message, Conn: conn}
					conn.handler.ServeRequest(m)
				}
			}
		}
	}

	return nil
}

func (conn *GrpcConnection) Handle(handler Handler) {
	conn.handler = handler
}

// TODO : implements me w/ test case
func (conn *GrpcConnection) verify(envelope *pb.Message) bool {
	return true
}

func (conn *GrpcConnection) isDie() bool {
	return atomic.LoadInt32(&(conn.stopFlag)) == int32(1)
}

func (conn *GrpcConnection) writeStream() {
	for !conn.isDie() {
		select {

		case m := <-conn.outChan:
			err := conn.streamWrapper.Send(m.Message)
			if err != nil {
				if m.OnErr != nil {
					go m.OnErr(err)
				}
			} else {
				if m.OnSuccess != nil {
					go m.OnSuccess("")
				}
			}
		case stop := <-conn.stopChan:
			conn.stopChan <- stop
			return
		}
	}
}

func (conn *GrpcConnection) readStream(errChan chan error) {
	defer func() {
		recover()
	}()

	for !conn.isDie() {
		envelope, err := conn.streamWrapper.Recv()

		if conn.isDie() {
			return
		}

		if err != nil {
			errChan <- err
			return
		}

		conn.readChan <- envelope
	}
}

type Broadcaster interface {
	Broadcast(data []byte, typ string, succCallback func(interface{}), errCallback func(error))
}

type ConnectionPool struct {
	connMap map[ConnID]Connection
}

func (p *ConnectionPool) Broadcast(data []byte, typ string) {
	panic("implement me w/ test case :-)")
}

func (p *ConnectionPool) Add(id ConnID, conn Connection) {
	panic("implement me w/ test case :-)")
}

func (p *ConnectionPool) Remove(id ConnID) {
	panic("implement me w/ test case :-)")
}
