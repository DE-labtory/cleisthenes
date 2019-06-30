package acs

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/acs"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/iLogger"
)

type handler struct {
	ServeRequestFunc func(msg cleisthenes.Message)
}

func newMockHandler() *handler {
	return &handler{}
}

func (h *handler) ServeRequest(msg cleisthenes.Message) {
	h.ServeRequestFunc(msg)
}

type NodeType int

const (
	Normal NodeType = iota
	Normal_No_Propose
	Normal_Latency
	Byzantine_Stupid
	Byzantine_Interceptor
)

type Node struct {
	lock sync.RWMutex

	n, f int
	typ  NodeType

	owner  cleisthenes.Member
	acs    *acs.ACS
	output map[cleisthenes.Member][]byte

	server        *cleisthenes.GrpcServer
	connPool      *cleisthenes.ConnectionPool
	memberMap     *cleisthenes.MemberMap
	batchReceiver cleisthenes.BatchReceiver

	doneChan chan struct{}
}

func NewNode(n, f int, owner cleisthenes.Member, batchReceiver cleisthenes.BatchReceiver, typ NodeType) *Node {
	server := cleisthenes.NewServer(owner.Address)
	connPool := cleisthenes.NewConnectionPool()
	memberMap := cleisthenes.NewMemberMap()

	return &Node{
		n:             n,
		f:             f,
		typ:           typ,
		lock:          sync.RWMutex{},
		owner:         owner,
		acs:           new(acs.ACS),
		server:        server,
		connPool:      connPool,
		memberMap:     memberMap,
		batchReceiver: batchReceiver,
		doneChan:      make(chan struct{}, n),
	}
}

func (n *Node) Run() {
	handler := newMockHandler()
	handler.ServeRequestFunc = func(msg cleisthenes.Message) {
		n.doAction(&msg)
		n.serveRequestFunc(msg)
	}

	n.server.OnConn(func(conn cleisthenes.Connection) {
		iLogger.Infof(nil, "[server] on connection from : %s", n.owner.Address.String())
		conn.Handle(handler)
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	})

	go n.server.Listen()
	go n.run()
}

func (n *Node) run() {
	for {
		select {
		case msg := <-n.batchReceiver.Receive():
			n.output = msg.Batch
			n.doneChan <- struct{}{}
		}
	}
}

func (n *Node) Connect(addr cleisthenes.Address) error {
	cli := cleisthenes.NewClient()
	conn, err := cli.Dial(cleisthenes.DialOpts{
		Addr: cleisthenes.Address{
			Ip:   addr.Ip,
			Port: addr.Port,
		},
		Timeout: cleisthenes.DefaultDialTimeout,
	})
	if err != nil {
		errors.New(fmt.Sprintf("fail to dial to : %s, with error: %s", addr.String(), err))
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

func (n *Node) Close() {
	n.server.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}
}

func (n *Node) Propose(data []byte) error {
	if err := n.acs.HandleInput(data); err != nil {
		return errors.New(fmt.Sprintf("error in HandleInput : %s", err.Error()))
	}

	return nil
}

func (n *Node) Address() cleisthenes.Address {
	return n.owner.Address
}

func (n *Node) serveRequestFunc(msg cleisthenes.Message) {
	senderAddr, _ := cleisthenes.ToAddress(msg.Sender)
	sender, _ := n.memberMap.Member(senderAddr)
	n.acs.HandleMessage(sender, msg.Message)
}

func (n *Node) doAction(msg *cleisthenes.Message) {
	switch n.typ {
	case Normal:
		// nothing
	case Normal_No_Propose:
		// nothing
	case Normal_Latency:
		n.doLatency(msg)
	case Byzantine_Stupid:
		n.doStupid()
	case Byzantine_Interceptor:
		n.doInterceptor(msg)
	}
}

func (n Node) doLatency(msg *cleisthenes.Message) {
	seed := rand.NewSource(msg.Timestamp.Seconds)
	random := rand.New(seed)
	interval := random.Intn(100)
	time.Sleep(time.Duration(interval) * time.Millisecond)
}

func (n Node) doStupid() {
	none := make(chan struct{}, 0)
	<-none
}

func (n Node) doInterceptor(msg *cleisthenes.Message) {
	switch msg.Payload.(type) {
	case *pb.Message_Rbc:
		msg.GetRbc().Payload = append(msg.GetRbc().Payload, []byte("invalid")...)
	case *pb.Message_Bba:
		msg.GetBba().Payload = append(msg.GetBba().Payload, byte(0x00))
	default:
		// nothing
	}
}
