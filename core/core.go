package core

import (
	"fmt"

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
}

type Node struct {
	addr            cleisthenes.Address
	txQueueManager  cleisthenes.TxQueueManager
	batchReceiver   cleisthenes.BatchReceiver
	messageEndpoint cleisthenes.MessageEndpoint
	server          *cleisthenes.GrpcServer
	client          *cleisthenes.GrpcClient
	connPool        *cleisthenes.ConnectionPool
	memberMap       *cleisthenes.MemberMap
	txValidator     cleisthenes.TxVerifyFunc
}

func New() (Hbbft, error) {
	conf := config.Get()

	addr, err := cleisthenes.ToAddress(conf.Identity.Address)
	if err != nil {
		return nil, err
	}

	memberMap := cleisthenes.NewMemberMap()
	memberMap.Add(&cleisthenes.Member{Address: addr})
	for _, addrStr := range conf.Members.Addresses {
		addr, err := cleisthenes.ToAddress(addrStr)
		if err != nil {
			return nil, err
		}
		memberMap.Add(&cleisthenes.Member{Address: addr})
	}

	txQueue := cleisthenes.NewTxQueue()
	hb := honeybadger.New()

	return &Node{
		addr: addr,
		txQueueManager: cleisthenes.NewDefaultTxQueueManager(
			txQueue,
			hb,
			conf.HoneyBadger.BatchSize,
			conf.HoneyBadger.BatchSize*len(memberMap.Members()),
			conf.HoneyBadger.ProposeInterval,
		),
		batchReceiver:   cleisthenes.NewBatchChannel(),
		messageEndpoint: hb,
		server:          cleisthenes.NewServer(addr),
		client:          cleisthenes.NewClient(),
		connPool:        cleisthenes.NewConnectionPool(),
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
	return n.txQueueManager.AddTransaction(tx, n.txValidator)
}

func (n *Node) Connect() error {
	for _, addrStr := range config.Get().Members.Addresses {
		addr, err := cleisthenes.ToAddress(addrStr)
		if err != nil {
			return err
		}
		if err := n.connect(addr); err != nil {
			return err
		}
	}

	return nil
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
	n.memberMap.Add(&cleisthenes.Member{Address: addr})
	return nil
}

func (n *Node) Close() {
	n.server.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}
}

func (n *Node) Result() <-chan cleisthenes.BatchMessage {
	return n.batchReceiver.Receive()
}
