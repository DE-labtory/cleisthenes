package acs

import (
	"fmt"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/acs"
	"github.com/DE-labtory/cleisthenes/test/util"
)

func setNodeList(n, f int, nodeType []NodeType) []*Node {
	var ip string = "127.0.0.1"
	var port uint16 = 8000

	batchChanList := make([]*cleisthenes.BatchChannel, 0)
	nodeList := make([]*Node, 0)
	for i := 0; i < n; i++ {
		availablePort := util.GetAvailablePort(port)
		owner := cleisthenes.NewMember(ip, availablePort)
		batchChan := cleisthenes.NewBatchChannel(1)
		batchChanList = append(batchChanList, batchChan)
		node := NewNode(n, f, *owner, batchChan, nodeType[i])
		node.Run()
		time.Sleep(50 * time.Millisecond)
		nodeList = append(nodeList, node)

		port = availablePort + 1
	}

	for m, mNode := range nodeList {

		// myself
		member := cleisthenes.NewMember(mNode.owner.Address.Ip, mNode.owner.Address.Port)
		mNode.memberMap.Add(member)
		for y, yNode := range nodeList {
			if m != y {
				mNode.Connect(yNode.Address())
			}
		}

		dataChan := cleisthenes.NewDataChannel(n)
		binaryChan := cleisthenes.NewBinaryChannel(n)

		a, _ := acs.New(n, f,
			0,
			mNode.owner,
			*mNode.memberMap,
			dataChan,
			dataChan,
			binaryChan,
			binaryChan,
			batchChanList[m],
			mNode.connPool)

		mNode.acs = a
	}

	return nodeList
}

func ACSTest(proposer *Node, data []byte) error {
	if proposer.typ == Normal {
		proposer.Propose(data)
	}

	for {
		select {
		case <-proposer.doneChan:
			fmt.Println("done")
			return nil
		}
	}
}

func Test_ACS_4NODEs(t *testing.T) {
	n := 4
	f := 1

	// Normal : 4, Proposer : 3
	nodeType := []NodeType{Normal, Normal, Normal, Normal_No_Propose}
	nodeList := setNodeList(n, f, nodeType)

	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			data := fmt.Sprintf("node %d", idx)
			if err := ACSTest(nodeList[idx], []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	for _, node := range nodeList {
		if len(node.output) != 3 {
			t.Fatalf("invalid output count - expected : %d, got : %d", n, len(node.output))
		}
	}

	// wait for late messages
	time.Sleep(100 * time.Millisecond)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_ACS_4NODEs_BYZANTINE_STUPID(t *testing.T) {
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Normal, Byzantine_Stupid}
	nodeList := setNodeList(n, f, nodeType)

	wg := sync.WaitGroup{}
	wg.Add(3)
	for i := 0; i < n; i++ {
		go func(idx int) {
			data := fmt.Sprintf("node %d", idx)
			if err := ACSTest(nodeList[idx], []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	for _, node := range nodeList[:3] {
		if len(node.output) != n-f {
			t.Fatalf("invalid output count - expected : %d, got : %d", n-f, len(node.output))
		}
	}

	// wait for late messages
	time.Sleep(100 * time.Millisecond)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_ACS_4NODEs_BYZANTINE_INTERCEPTOR(t *testing.T) {
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Normal, Byzantine_Interceptor}
	nodeList := setNodeList(n, f, nodeType)

	wg := sync.WaitGroup{}
	wg.Add(3)
	for i := 0; i < n; i++ {
		go func(idx int) {
			data := fmt.Sprintf("node %d", idx)
			if err := ACSTest(nodeList[idx], []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	for _, node := range nodeList[:3] {
		if len(node.output) != n-f {
			t.Fatalf("invalid output count - expected : %d, got : %d", n-f, len(node.output))
		}
	}

	// wait for late messages
	time.Sleep(100 * time.Millisecond)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_ACS_4NODEs_BYZANTINE_FAULT(t *testing.T) {
	// arbitrary value for network timeout to identify byzantine scenario
	const TIMEOUT = 3 * time.Second
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Byzantine_Stupid, Byzantine_Stupid}
	nodeList := setNodeList(n, f, nodeType)
	doneChan := make(chan bool, len(nodeList))

	for i := 0; i < n; i++ {
		go func(idx int) {
			data := fmt.Sprintf("node %d", idx)
			if err := ACSTest(nodeList[idx], []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			doneChan <- true
		}(i)
	}

	time.Sleep(TIMEOUT)

	if len(doneChan) != 0 {
		t.Fatalf("test fail! it can't tolerate")
	}

	// wait for late messages
	time.Sleep(100 * time.Millisecond)
	for _, node := range nodeList {
		node.Close()
	}
}
