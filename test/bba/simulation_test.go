package bba_test

import (
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/iLogger"

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
		iLogger.Infof(nil, "addr: %s, value: %v", out.addr, out.value)
	}
}

type output struct {
	addr  cleisthenes.Address
	value cleisthenes.Binary
}

func TestBBA(t *testing.T) {
	n := 4
	f := 1
	nodeList := make([]*bba.Node, 0)

	for i := 0; i < n; i++ {
		port := util.GetAvailablePort(8000)
		node, err := bba.New(n, f, cleisthenes.Address{
			Ip:   "127.0.0.1",
			Port: port,
		})
		if err != nil {
			t.Fatalf("failed to create bba node")
		}
		nodeList = append(nodeList, node)
		node.Run()
		time.Sleep(100 * time.Millisecond)
	}

	for i, node := range nodeList {
		for ii, target := range nodeList {
			if ii == i {
				continue
			}
			if err := node.Connect(target.Info()); err != nil {
				t.Fatalf("failed to connect to target node: %s", target.Info().String())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	for i, node := range nodeList {
		if err := node.Propose(cleisthenes.One); err != nil {
			t.Fatalf("node propose binary failed with err: err=%s, node=%d", err, i)
		}
	}

	simulationResult := newSimulationResult()
	for {
		for _, node := range nodeList {
			result := node.Result()
			if !result.Undefined() && !simulationResult.contain(node.Info()) {
				simulationResult.append(output{
					addr: node.Info(), value: result.Value()})
			}

		}
		if len(simulationResult.outputList) == n {
			break
		}
	}

	simulationResult.print()
}
