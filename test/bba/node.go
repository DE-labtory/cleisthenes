package bba

import (
	"time"

	"github.com/DE-labtory/cleisthenes"
	engine "github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/iLogger"
)

type NodeType string

const (
	Normal = "normal"
	Lazy   = "lazy"
)

func normalNodeHandler(n *Node) func(msg cleisthenes.Message) {
	return func(msg cleisthenes.Message) {
		bbaMessage, ok := msg.Message.Payload.(*pb.Message_Bba)
		if !ok {
			iLogger.Fatalf(nil, "[handler] received message is not Message_Bba type")
		}
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			iLogger.Fatalf(nil, "[handler] failed to parse sender address: addr=%s", addr)
		}
		n.bba.HandleMessage(n.memberMap.Member(addr), bbaMessage)
	}
}
func lazyNodeHandler(n *Node) func(msg cleisthenes.Message) {
	return func(msg cleisthenes.Message) {
		// lazy node sleep 2 seconds
		time.Sleep(2000)

		bbaMessage, ok := msg.Message.Payload.(*pb.Message_Bba)
		if !ok {
			iLogger.Fatalf(nil, "[handler] received message is not Message_Bba type")
		}
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			iLogger.Fatalf(nil, "[handler] failed to parse sender address: addr=%s", addr)
		}
		n.bba.HandleMessage(n.memberMap.Member(addr), bbaMessage)
	}
}

type handler struct {
	handleFunc func(msg cleisthenes.Message)
}

func newHandler(handleFunc func(cleisthenes.Message)) *handler {
	return &handler{
		handleFunc: handleFunc,
	}
}

func (h *handler) ServeRequest(msg cleisthenes.Message) {
	h.handleFunc(msg)
}

type Node struct {
	typ       NodeType
	addr      cleisthenes.Address
	bba       *engine.BBA
	server    *cleisthenes.GrpcServer
	client    *cleisthenes.GrpcClient
	connPool  *cleisthenes.ConnectionPool
	memberMap *cleisthenes.MemberMap
}

func New(typ NodeType, n, f int, coinGenerator cleisthenes.CoinGenerator, addr cleisthenes.Address) (*Node, error) {
	member := &cleisthenes.Member{Address: addr}
	connPool := cleisthenes.NewConnectionPool()
	memberMap := cleisthenes.NewMemberMap()
	binChan := cleisthenes.NewBinaryChannel(n)
	bba := engine.New(n, f, *member, cleisthenes.Member{}, connPool, coinGenerator, binChan)
	return &Node{
		typ:       typ,
		addr:      addr,
		bba:       bba,
		server:    cleisthenes.NewServer(addr),
		client:    cleisthenes.NewClient(),
		connPool:  connPool,
		memberMap: memberMap,
	}, nil
}

func (n *Node) Run() {
	var handlerFunc func(message cleisthenes.Message)
	switch n.typ {
	case Normal:
		handlerFunc = normalNodeHandler(n)
	case Lazy:
		handlerFunc = lazyNodeHandler(n)
	}
	handler := newHandler(handlerFunc)

	n.server.OnConn(func(conn cleisthenes.Connection) {
		conn.Handle(handler)

		if err := conn.Start(); err != nil {
			conn.Close()
		}
	})

	go n.server.Listen()
}

func (n *Node) Connect(addr cleisthenes.Address) error {
	conn, err := n.client.Dial(cleisthenes.DialOpts{
		Addr:    addr,
		Timeout: cleisthenes.DefaultDialTimeout,
	})
	if err != nil {
		return err
	}
	go func() {
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	}()

	n.connPool.Add(addr, conn)
	n.memberMap.Add(&cleisthenes.Member{Address: addr})

	return nil
}

func (n *Node) Propose(bin cleisthenes.Binary) error {
	bvalRequest := &engine.BvalRequest{Value: bin}
	return n.bba.HandleInput(bvalRequest)
}

func (n *Node) Close() {
	n.server.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}
}

func (n *Node) Info() cleisthenes.Address {
	return n.addr
}

func (n *Node) Result() (cleisthenes.Binary, bool) {
	return n.bba.Result()
}

func (n *Node) Trace() {
	n.bba.Trace()
}
