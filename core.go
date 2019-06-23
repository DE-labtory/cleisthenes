package cleisthenes

import (
	"github.com/DE-labtory/cleisthenes/config"
)

type Hbbft interface {
	Propose(tx Transaction) error
	Run()
}

type Node struct {
	addr           Address
	txQueueManager TxQueueManager
	server         *GrpcServer
	client         *GrpcClient
	connPool       *ConnectionPool
	memberMap      *MemberMap
}

func New() (Hbbft, error) {
	conf := config.Get()
	addr, err := ToAddress(conf.Identity.Address)
	if err != nil {
		return nil, err
	}
	return &Node{
		server:    NewServer(addr),
		client:    NewClient(),
		connPool:  NewConnectionPool(),
		memberMap: NewMemberMap(),
	}, nil
}

func (n *Node) Run() {

}

func (n *Node) Propose(tx Transaction) error { return nil }

func (n *Node) Connect() error { return nil }

func (n *Node) Close() {}
