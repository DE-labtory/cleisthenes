package mock

import (
	"github.com/DE-labtory/cleisthenes"
)

var Coin = cleisthenes.Coin(cleisthenes.One)

type CoinGenerator struct{}

func (g *CoinGenerator) Coin() cleisthenes.Coin {
	return Coin
}
