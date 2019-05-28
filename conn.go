package cleisthenes

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnId = string

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
	Id() ConnId
	Close()
	Start() error
	Handle(handler Handler)
}

type GrpcConnection struct {
	id            ConnId
	ip            Address
	streamWrapper StreamWrapper
	stopFlag      int32
	handler       Handler
	outChan       chan *innerMessage
	readChan      chan *pb.Message
	stopChan      chan struct{}
	sync.RWMutex
}

func NewConnection(ip Address, id ConnId, streamWrapper StreamWrapper) (Connection, error) {
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

func (conn *GrpcConnection) Id() ConnId {
	return conn.id
}

func (conn *GrpcConnection) Close() {
	if conn.isDie() {
		return
	}

	isFirst := atomic.CompareAndSwapInt32(&conn.stopFlag, int32(0), int32(1))
	if !isFirst {
		return
	}

	conn.stopChan <- struct{}{}
	conn.Lock()
	defer conn.Unlock()

	conn.streamWrapper.Close()
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
	// ShareMessage is a function that broadcast Single message to all nodes
	ShareMessage(msg pb.Message)

	// DistributeMessage is a function that broadcast Multiple messages one by one to all nodes
	DistributeMessage(msgList []pb.Message)
}

type ConnectionPool struct {
	connMap map[ConnId]Connection
	lock    sync.RWMutex
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connMap: make(map[ConnId]Connection),
		lock:    sync.RWMutex{},
	}
}

func (p *ConnectionPool) GetAll() []Connection {
	p.lock.Lock()
	defer p.lock.Unlock()

	connList := make([]Connection, 0)
	for _, conn := range p.connMap {
		connList = append(connList, conn)
	}
	return connList
}

func (p *ConnectionPool) ShareMessage(msg pb.Message) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, conn := range p.GetAll() {
		conn.Send(msg, nil, nil)
	}
}

func (p *ConnectionPool) DistributeMessage(msgList []pb.Message) {
	for i, conn := range p.GetAll() {
		if i == len(msgList) {
			break
		}
		conn.Send(msgList[i], nil, nil)
	}
}

func (p *ConnectionPool) Add(id ConnId, conn Connection) {
	p.connMap[id] = conn
}

func (p *ConnectionPool) Remove(id ConnId) {
	delete(p.connMap, id)
}
