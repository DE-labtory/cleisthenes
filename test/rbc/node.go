package rbc

import (
	"errors"
	"fmt"
	"sync"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/rbc"
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

type mockRbcOutputMap struct {
	lock  sync.RWMutex
	items map[cleisthenes.Address][]byte
}

func newMockRbcOutputMap(size int) *mockRbcOutputMap {
	return &mockRbcOutputMap{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Address][]byte, size),
	}
}

func (m *mockRbcOutputMap) set(addr cleisthenes.Address, data []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items[addr] = data
}

func (m *mockRbcOutputMap) item(addr cleisthenes.Address) []byte {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.items[addr]
}

func (m *mockRbcOutputMap) delete(addr cleisthenes.Address) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items[addr] = nil
}

type NodeType int

const (
	Normal NodeType = iota
	Byzantine_Stupid
	Byzantine_Interceptor
)

type Node struct {
	n, f int
	typ  NodeType
	lock sync.RWMutex

	// node's address
	address cleisthenes.Address
	rbcMap  map[cleisthenes.Address]*rbc.RBC
	server  *cleisthenes.GrpcServer

	// connPool has only other nodes (except me)
	connPool *cleisthenes.ConnectionPool

	// memberMap include proposer node
	memberMap     *cleisthenes.MemberMap
	dataReceiver  cleisthenes.DataReceiver
	dataOutputMap *mockRbcOutputMap

	doneChan chan struct{}
}

func NewNode(n, f int, addr cleisthenes.Address, dataReceiver cleisthenes.DataReceiver, typ NodeType) *Node {
	server := cleisthenes.NewServer(addr)
	connPool := cleisthenes.NewConnectionPool()
	memberMap := cleisthenes.NewMemberMap()

	return &Node{
		n:             n,
		f:             f,
		typ:           typ,
		lock:          sync.RWMutex{},
		rbcMap:        make(map[cleisthenes.Address]*rbc.RBC, 0),
		address:       addr,
		server:        server,
		connPool:      connPool,
		memberMap:     memberMap,
		dataReceiver:  dataReceiver,
		dataOutputMap: newMockRbcOutputMap(n),
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
		iLogger.Infof(nil, "[server] on connection from : %s", n.address.String())
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
		case msg := <-n.dataReceiver.Receive():
			n.dataOutputMap.set(msg.Member.Address, msg.Data)
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
	if err := n.rbcMap[n.address].HandleInput(data); err != nil {
		return errors.New(fmt.Sprintf("error in MakeRequest : %s", err.Error()))
	}

	return nil
}

func (n *Node) Address() cleisthenes.Address {
	return n.address
}

func (n *Node) serveRequestFunc(msg cleisthenes.Message) {
	senderAddr, err := cleisthenes.ToAddress(msg.Sender)
	if err != nil {
		iLogger.Fatalf(nil, "[HANDLER] invalid sender address : %s", msg.Sender)
	}
	sender := n.memberMap.Member(senderAddr)

	proposerAddr, err := cleisthenes.ToAddress(msg.GetRbc().Proposer)
	if err != nil {
		iLogger.Fatalf(nil, "[HANDLER] invalid proposer address : %s", msg.GetRbc().Proposer)
	}
	r := n.rbcMap[proposerAddr]
	if err := r.HandleMessage(sender, &pb.Message_Rbc{
		Rbc: msg.GetRbc(),
	}); err != nil {
		iLogger.Errorf(nil, "error in Handlemessage : %s", err.Error())
	}
}

func (n *Node) doAction(msg *cleisthenes.Message) {
	switch n.typ {
	case Normal:
		// nothing
	case Byzantine_Stupid:
		n.doStupid()
	case Byzantine_Interceptor:
		n.doInterceptor(msg)
	}
}

func (n Node) doStupid() {
	none := make(chan struct{}, 0)
	<-none
}

func (n Node) doInterceptor(msg *cleisthenes.Message) {
	msg.GetRbc().Payload = append(msg.GetRbc().Payload, []byte("invalid")...)
}
