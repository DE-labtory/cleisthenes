package honeybadger

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/honeybadger"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/iLogger"
)

// 하나로 빼기
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

type Batch map[cleisthenes.Member][]byte

type Node struct {
	n, f int
	typ  NodeType

	owner cleisthenes.Member

	hb             *honeybadger.HoneyBadger
	server         *cleisthenes.GrpcServer
	connPool       *cleisthenes.ConnectionPool
	memberMap      *cleisthenes.MemberMap
	resultBatch    map[cleisthenes.Epoch]Batch
	resultReceiver cleisthenes.ResultReceiver

	doneChan chan struct{}
}

func NewNode(n, f int, owner cleisthenes.Member, resultReceiver cleisthenes.ResultReceiver, typ NodeType) *Node {
	server := cleisthenes.NewServer(owner.Address)
	connPool := cleisthenes.NewConnectionPool()
	memberMap := cleisthenes.NewMemberMap()

	return &Node{
		n:              n,
		f:              f,
		typ:            typ,
		owner:          owner,
		server:         server,
		resultBatch:    make(map[cleisthenes.Epoch]Batch, 0),
		hb:             new(honeybadger.HoneyBadger),
		connPool:       connPool,
		memberMap:      memberMap,
		resultReceiver: resultReceiver,
		doneChan:       make(chan struct{}, n),
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
		case <-n.resultReceiver.Receive():
			//n.resultBatch[msg.Epoch] = msg.Batch
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

func (n *Node) Propose(contribution cleisthenes.Contribution) {
	n.hb.HandleContribution(contribution)
}

func (n *Node) Address() cleisthenes.Address {
	return n.owner.Address
}

func (n *Node) serveRequestFunc(msg cleisthenes.Message) {
	if err := n.hb.HandleMessage(msg.Message); err != nil {
		fmt.Println("[ERR]", err.Error())
	}
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
