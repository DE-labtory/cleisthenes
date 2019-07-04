package cleisthenes

type Coin Binary

type CoinGenerator interface {
	Coin(uint64) Coin
}
