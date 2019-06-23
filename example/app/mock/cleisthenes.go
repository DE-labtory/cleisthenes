package mock

import (
	"errors"

	"github.com/DE-labtory/cleisthenes/core"

	"github.com/DE-labtory/cleisthenes/config"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/example/app"
	"github.com/go-kit/kit/log"
)

type Node struct {
	logger log.Logger
}

func NewMockNode(logger log.Logger) core.Hbbft {
	return &Node{logger: logger}
}

func (n *Node) Submit(tx cleisthenes.Transaction) error {
	transaction, ok := tx.(app.Transaction)
	if !ok {
		return errors.New("invalid transaction type")
	}
	n.logger.Log(
		"message", "transaction proposed",
		"from", transaction.From,
		"to", transaction.To,
		"amount", transaction.Amount,
	)
	return nil
}

func (n *Node) Run() {
	conf := config.Get()
	n.logger.Log("network_size", conf.HoneyBadger.NetworkSize)
	n.logger.Log("byzantine_size", conf.HoneyBadger.Byzantine)
}
