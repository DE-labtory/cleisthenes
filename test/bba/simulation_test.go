package bba_test

import (
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes/log"

	"github.com/DE-labtory/cleisthenes/test/mock"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/test/bba"
	"github.com/DE-labtory/cleisthenes/test/util"
)

type simulationResult struct {
	lock       sync.RWMutex
	outputList []output
}

func newSimulationResult() *simulationResult {
	return &simulationResult{
		lock:       sync.RWMutex{},
		outputList: make([]output, 0),
	}
}

func (r *simulationResult) contain(addr cleisthenes.Address) bool {
	for _, out := range r.outputList {
		if out.addr.String() == addr.String() {
			return true
		}
	}
	return false
}

func (r *simulationResult) append(out output) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.outputList = append(r.outputList, out)
}

func (r *simulationResult) print() {
	for _, out := range r.outputList {
		log.Info("action", "result", "addr", out.addr, "value", out.value)
	}
}

type output struct {
	addr  cleisthenes.Address
	value cleisthenes.Binary
}

func runNode(t *testing.T, n, f int, coinGeneratorList []cleisthenes.CoinGenerator) []*bba.Node {
	nodeList := make([]*bba.Node, 0)
	for i := 0; i < n; i++ {
		port := util.GetAvailablePort(8000)
		node, err := bba.New(
			n,
			f,
			coinGeneratorList[i],
			cleisthenes.Address{
				Ip:   "127.0.0.1",
				Port: port,
			})
		if err != nil {
			t.Fatalf("failed to create bba node")
		}
		nodeList = append(nodeList, node)
		node.Run()
		time.Sleep(50 * time.Millisecond)
	}
	return nodeList
}

func connectNode(t *testing.T, nodeList []*bba.Node) {
	for i, node := range nodeList {
		for ii, target := range nodeList {
			if ii == i {
				continue
			}
			if err := node.Connect(target.Info()); err != nil {
				t.Fatalf("failed to connect to target node: %s", target.Info().String())
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func proposeBinary(t *testing.T, nodeList []*bba.Node, binList []cleisthenes.Binary) {
	for i, node := range nodeList {
		if err := node.Propose(binList[i]); err != nil {
			t.Fatalf("node propose binary failed with err: err=%s, node=%d", err, i)
		}
	}
}

func TestBBA_WithoutByzantine(t *testing.T) {
	n := 4
	f := 1

	binList := []cleisthenes.Binary{
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
	}
	coinGeneratorList := []cleisthenes.CoinGenerator{
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
	}

	nodeList := runNode(t, n, f, coinGeneratorList)

	connectNode(t, nodeList)

	proposeBinary(t, nodeList, binList)

	result := watchResult(nodeList)

	assertResult(t, result, cleisthenes.One)
}

func TestBBA_WithByzantine(t *testing.T) {
	n := 10
	f := 3

	binList := []cleisthenes.Binary{
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.One,
		cleisthenes.Zero,
		cleisthenes.Zero,
		cleisthenes.Zero,
	}
	coinGeneratorList := []cleisthenes.CoinGenerator{
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
	}

	nodeList := runNode(t, n, f, coinGeneratorList)

	connectNode(t, nodeList)

	proposeBinary(t, nodeList, binList)

	result := watchResult(nodeList)

	assertResult(t, result, cleisthenes.One)
}

func watchResult(nodeList []*bba.Node) *simulationResult {
	simulationResult := newSimulationResult()
	for {
		for _, node := range nodeList {
			result, ok := node.Result()
			if !ok && !simulationResult.contain(node.Info()) {
				simulationResult.append(output{
					addr: node.Info(), value: result})
			}
		}
		if len(simulationResult.outputList) == len(nodeList) {
			break
		}
	}

	simulationResult.print()

	return simulationResult
}

func assertResult(t *testing.T, result *simulationResult, expected cleisthenes.Binary) {
	for _, output := range result.outputList {
		if output.value != expected {
			t.Fatalf("%s: expected value is %v, but got %v", output.addr.String(), expected, output.value)
		}
	}
}
