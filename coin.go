package cleisthenes

type Coin Binary

type CoinGenerator interface {
	Coin() Coin
}
