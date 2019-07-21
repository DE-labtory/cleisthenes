package core

import (
	"fmt"

	"github.com/DE-labtory/cleisthenes/tpke"

	"github.com/DE-labtory/cleisthenes"

	"github.com/DE-labtory/cleisthenes/config"
	"github.com/DE-labtory/cleisthenes/honeybadger"
)

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

type Hbbft interface {
	Submit(tx cleisthenes.Transaction) error
	Run()
	Connect(target string) error
	ConnectAll(targetList []string) error
	ConnectionList() []string
	Result() <-chan cleisthenes.ResultMessage
}

type Node struct {
	addr            cleisthenes.Address
	txQueueManager  cleisthenes.TxQueueManager
	resultReceiver  cleisthenes.ResultReceiver
	messageEndpoint cleisthenes.MessageEndpoint
	server          *cleisthenes.GrpcServer
	client          *cleisthenes.GrpcClient
	connPool        *cleisthenes.ConnectionPool
	memberMap       *cleisthenes.MemberMap
	txValidator     cleisthenes.TxValidator
}

func New(txValidator cleisthenes.TxValidator) (Hbbft, error) {
	conf := config.Get()

	addr, err := cleisthenes.ToAddress(conf.Identity.Address)
	if err != nil {
		return nil, err
	}

	memberMap := cleisthenes.NewMemberMap()
	memberMap.Add(cleisthenes.NewMemberWithAddress(addr))

	connPool := cleisthenes.NewConnectionPool()

	dataChan := cleisthenes.NewDataChannel(conf.HoneyBadger.NetworkSize)
	batchChan := cleisthenes.NewBatchChannel(conf.HoneyBadger.NetworkSize)
	binChan := cleisthenes.NewBinaryChannel(conf.HoneyBadger.NetworkSize)
	resultChan := cleisthenes.NewResultChannel(conf.HoneyBadger.NetworkSize)

	txQueue := cleisthenes.NewTxQueue()
	hb := honeybadger.New(
		memberMap,
		honeybadger.NewDefaultACSFactory(
			conf.HoneyBadger.NetworkSize,
			conf.HoneyBadger.Byzantine,
			*cleisthenes.NewMemberWithAddress(addr),
			*memberMap,
			dataChan,
			dataChan,
			binChan,
			binChan,
			batchChan,
			connPool,
		),
		&tpke.MockTpke{},
		batchChan,
		resultChan,
	)

	return &Node{
		addr: addr,
		txQueueManager: cleisthenes.NewDefaultTxQueueManager(
			txQueue,
			hb,
			conf.HoneyBadger.BatchSize/len(memberMap.Members()),
			// TODO : contribution size = B / N
			conf.HoneyBadger.BatchSize,
			conf.HoneyBadger.ProposeInterval,
			txValidator,
		),
		resultReceiver:  resultChan,
		messageEndpoint: hb,
		server:          cleisthenes.NewServer(addr),
		client:          cleisthenes.NewClient(),
		connPool:        connPool,
		memberMap:       cleisthenes.NewMemberMap(),
	}, nil
}

func (n *Node) Run() {
	handler := newHandler(func(msg cleisthenes.Message) {
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			fmt.Println(fmt.Errorf("[handler] failed to parse sender address: addr=%s", addr))
		}
		if err := n.messageEndpoint.HandleMessage(msg.Message); err != nil {
			fmt.Println(fmt.Errorf("[handler] handle message failed with err: %s", err.Error()))
		}
	})

	n.server.OnConn(func(conn cleisthenes.Connection) {
		conn.Handle(handler)
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	})

	go n.server.Listen()
}

func (n *Node) Submit(tx cleisthenes.Transaction) error {
	return n.txQueueManager.AddTransaction(tx)
}

func (n *Node) ConnectAll(targetList []string) error {
	for _, target := range targetList {
		if err := n.Connect(target); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Connect(target string) error {
	addr, err := cleisthenes.ToAddress(target)
	if err != nil {
		return err
	}
	return n.connect(addr)
}

func (n *Node) connect(addr cleisthenes.Address) error {
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
	n.memberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	return nil
}

func (n *Node) ConnectionList() []string {
	result := make([]string, 0)
	for _, conn := range n.connPool.GetAll() {
		result = append(result, conn.Ip().String())
	}
	return result
}

func (n *Node) Close() {
	n.server.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}
}

func (n *Node) Result() <-chan cleisthenes.ResultMessage {
	return n.resultReceiver.Receive()
}
