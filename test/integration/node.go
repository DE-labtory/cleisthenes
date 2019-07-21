package integration

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/iLogger"
	"github.com/mitchellh/mapstructure"
)

type exTransaction struct {
	Amount int
}

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

// TODO : data race check! remove lock property
type resultBatch struct {
	lock  sync.RWMutex
	txMap map[exTransaction]bool
}

func newResultBatch() resultBatch {
	return resultBatch{
		lock:  sync.RWMutex{},
		txMap: make(map[exTransaction]bool),
	}
}

func (r *resultBatch) add(tx exTransaction) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.txMap[tx] = true
}

func (r *resultBatch) len() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	len := len(r.txMap)
	return len
}

type Batch map[cleisthenes.Member][]byte

type Node struct {
	n, f int
	typ  NodeType

	owner cleisthenes.Member

	txQueueManager  cleisthenes.TxQueueManager
	messageEndpoint cleisthenes.MessageEndpoint
	txValidator     cleisthenes.TxValidator

	server         *cleisthenes.GrpcServer
	connPool       *cleisthenes.ConnectionPool
	memberMap      *cleisthenes.MemberMap
	resultBatch    resultBatch
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
		resultBatch:    newResultBatch(),
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
		case msg := <-n.resultReceiver.Receive():
			n.addToResultSet(msg.Batch)
			n.doneChan <- struct{}{}
		}
	}
}

func (n *Node) decrypt(tx cleisthenes.AbstractTx) exTransaction {
	var exTransaction exTransaction
	mapstructure.Decode(tx, &exTransaction)
	return exTransaction
}

func (n *Node) addToResultSet(txList []cleisthenes.AbstractTx) {
	for _, tx := range txList {
		exTx := n.decrypt(tx)
		n.resultBatch.add(exTx)
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

func (n *Node) AddTransaction(tx cleisthenes.Transaction) {
	n.txQueueManager.AddTransaction(tx)
}

func (n *Node) Address() cleisthenes.Address {
	return n.owner.Address
}

func (n *Node) serveRequestFunc(msg cleisthenes.Message) {
	if err := n.messageEndpoint.HandleMessage(msg.Message); err != nil {
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
