package mock

import (
	"github.com/DE-labtory/cleisthenes"
)

var (
	Coin = cleisthenes.Coin(cleisthenes.One)
)

type CoinGenerator struct {
	Seed cleisthenes.Coin
}

func NewCoinGenerator(seed cleisthenes.Coin) *CoinGenerator {
	return &CoinGenerator{
		Seed: seed,
	}
}

func (g *CoinGenerator) Coin(salt uint64) cleisthenes.Coin {
	return cleisthenes.Coin(salt%2 != 0)
}
