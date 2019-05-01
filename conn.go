package cleisthenes

import (
	"sync"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnID = string

// message used in HBBFT
type innerMessage struct {
	*pb.Message
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
	Send(data []byte, typ string, successCallBack func(interface{}), errCallBack func(error))
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
	outChan       chan *innerMessage
	readChan      chan *pb.Message
	stopChan      chan struct{}
	sync.RWMutex
}

func NewConnection(streamWrapper StreamWrapper) (Connection, error) {
	panic("implement me w/ test case :-)")
}

func (conn *GrpcConnection) Send(data []byte, typ string, successCallBack func(interface{}), errCallBack func(error)) {
	panic("implement me w/ test case :-)")
}

func (conn *GrpcConnection) Ip() string {
	panic("implement me w/ test case :-)")
}

func (conn *GrpcConnection) Id() string {
	panic("implement me w/ test case :-)")
}

func (conn *GrpcConnection) Close() {
	panic("implement me w/ test case :-)")
}

func (conn *GrpcConnection) Start() error {
	panic("implement me w/ test case :-)")
}

func (conn *GrpcConnection) Handle() {
	panic("implement me w/ test case :-)")
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
