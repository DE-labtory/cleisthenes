package bba

import (
	"encoding/json"
	"log"

	"github.com/DE-labtory/cleisthenes"
	engine "github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/test/mock"
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
	connPool := cleisthenes.NewConnectionPool()
	member := cleisthenes.Member{
		Address: addr,
	}
	bba := engine.New(n, f, initialEpoch, member, connPool, &mock.CoinGenerator{})
	return &Node{
		addr:       addr,
		bba:        bba,
		grpcServer: cleisthenes.NewServer(addr),
		grpcClient: cleisthenes.NewClient(),
		connPool:   connPool,
		memberMap:  cleisthenes.NewMemberMap(),
	}, nil
}

func (n *Node) Run() {
	handler := newHandler(func(msg cleisthenes.Message) {
		bbaMessage, ok := msg.Message.Payload.(*pb.Message_Bba)
		if !ok {
			log.Fatalf("[handler] received message is not Message_Bba type")
		}
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			log.Fatalf("[handler] failed to parse sender address: addr=%s", addr)
		}
		n.bba.HandleMessage(n.memberMap.Member(addr), bbaMessage)
	})

	n.grpcServer.OnConn(func(conn cleisthenes.Connection) {
		log.Println("[server] on connection")

		n.connPool.Add(conn.Ip(), conn)

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

	n.connPool.Add(addr, conn)
	return nil
}

func (n *Node) Propose(bin cleisthenes.Binary) error {
	bvalRequest := &engine.BvalRequest{Value: bin}
	data, err := json.Marshal(bvalRequest)
	if err != nil {
		return err
	}
	return n.bba.HandleInput(&pb.Message_Bba{
		Bba: &pb.BBA{
			Payload: data,
			Round:   1,
		},
	})
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
