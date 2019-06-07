package bba

import (
	"github.com/DE-labtory/cleisthenes"
	engine "github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/test/mock"
	"github.com/DE-labtory/iLogger"
)

const initialEpoch = 1

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
	addr       cleisthenes.Address
	bba        *engine.BBA
	grpcServer *cleisthenes.GrpcServer
	grpcClient *cleisthenes.GrpcClient
	connPool   *cleisthenes.ConnectionPool
	memberMap  *cleisthenes.MemberMap
}

func New(n, f int, addr cleisthenes.Address) (*Node, error) {
	member := &cleisthenes.Member{Address: addr}
	connPool := cleisthenes.NewConnectionPool()
	memberMap := cleisthenes.NewMemberMap()
	bba := engine.New(n, f, initialEpoch, *member, connPool, &mock.CoinGenerator{})
	return &Node{
		addr:       addr,
		bba:        bba,
		grpcServer: cleisthenes.NewServer(addr),
		grpcClient: cleisthenes.NewClient(),
		connPool:   connPool,
		memberMap:  memberMap,
	}, nil
}

func (n *Node) Run() {
	handler := newHandler(func(msg cleisthenes.Message) {
		iLogger.Infof(nil, "action: handling message, owner: %s, from: %s", n.Info().String(), msg.Sender)
		bbaMessage, ok := msg.Message.Payload.(*pb.Message_Bba)
		if !ok {
			iLogger.Fatalf(nil, "[handler] received message is not Message_Bba type")
		}
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			iLogger.Fatalf(nil, "[handler] failed to parse sender address: addr=%s", addr)
		}
		n.bba.HandleMessage(n.memberMap.Member(addr), bbaMessage)
	})

	n.grpcServer.OnConn(func(conn cleisthenes.Connection) {
		iLogger.Infof(nil, "server: on connection, from: %s", n.Info())
		conn.Handle(handler)

		if err := conn.Start(); err != nil {
			conn.Close()
		}
	})

	go n.grpcServer.Listen()
}

func (n *Node) Connect(addr cleisthenes.Address) error {
	conn, err := n.grpcClient.Dial(cleisthenes.DialOpts{
		Addr:    addr,
		Timeout: cleisthenes.DefaultDialTimeout,
	})
	if err != nil {
		return err
	}
	go func() {
		iLogger.Infof(nil, "client: connection start, from: %s", n.Info())
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
	n.grpcServer.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}
}

func (n *Node) Info() cleisthenes.Address {
	return n.addr
}

func (n *Node) Result() cleisthenes.BinaryState {
	return n.bba.Result()
}
