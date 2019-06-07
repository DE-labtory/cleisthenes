package bba_test

import (
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/test/bba"
	"github.com/DE-labtory/cleisthenes/test/util"
)

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
		time.Sleep(10 * time.Millisecond)
	}

	for i, node := range nodeList {
		for ii, target := range nodeList {
			if ii == i {
				continue
			}
			if err := node.Connect(target.Info()); err != nil {
				t.Fatalf("failed to connect to target node: %s", target.Info().String())
			}
		}
	}

	for i, node := range nodeList {
		if err := node.Propose(cleisthenes.One); err != nil {
			t.Fatalf("node propose binary failed with err: err=%s, node=%d", err, i)
		}
	}

	for {
		outputList := make([]cleisthenes.BinaryState, 0)
		for _, node := range nodeList {
			result := node.Result()
			if !result.Undefined() {
				outputList = append(outputList, result)
			}

		}
		if len(outputList) == n {
			break
		}
	}
}
