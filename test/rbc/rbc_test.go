package rbc

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes/test/util"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/rbc"
)

func setNodeList(n, f int, nodeType []NodeType) []*Node {
	var ip string = "127.0.0.1"
	var port uint16 = 8000

	nodeList := make([]*Node, 0)
	dataChanList := make([]*cleisthenes.DataChannel, 0)
	for i := 0; i < n; i++ {
		availablePort := util.GetAvailablePort(port)
		address := cleisthenes.Address{
			Ip:   ip,
			Port: availablePort,
		}
		dataChan := cleisthenes.NewDataChannel(n)
		dataChanList = append(dataChanList, dataChan)

		node := NewNode(n, f, address, dataChan, nodeType[i])
		node.Run()
		time.Sleep(50 * time.Millisecond)
		nodeList = append(nodeList, node)

		port = availablePort + 1
	}

	for m, mNode := range nodeList {
		member := cleisthenes.NewMember(mNode.address.Ip, mNode.address.Port)
		mNode.memberMap.Add(member)
		for y, yNode := range nodeList {
			if m != y {
				mNode.Connect(yNode.address)
			}
		}

		owner, _ := mNode.memberMap.Member(mNode.address)
		for _, member := range mNode.memberMap.Members() {
			r, _ := rbc.New(n, f, 0, owner, member, mNode.connPool, dataChanList[m])
			mNode.rbcMap[member.Address] = r
			mNode.dataOutputMap.set(member.Address, nil)
		}
	}

	return nodeList
}

func RBCTest(proposer *Node, nodeList []*Node, data []byte) error {
	nodeCnt := len(nodeList)
	byzantineCnt := 0
	for _, node := range nodeList {
		if node.typ != Normal {
			byzantineCnt++
		}
	}

	proposer.Propose(data)

	// check whether node's rbc done
	flagChan := make(chan struct{}, nodeCnt)
	for _, node := range nodeList {
		go checkResult(node, flagChan)
	}

	doneCnt := 0
	for {
		select {
		case <-flagChan:
			doneCnt++
			if doneCnt == nodeCnt-byzantineCnt {
				return nil
			}
		}
	}

	return nil
}

func checkResult(node *Node, flagChan chan struct{}) {
	for {
		select {
		case <-node.doneChan:
			flagChan <- struct{}{}
		}
	}
}

func Test_RBC_4NODEs(t *testing.T) {
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Normal, Normal}
	nodeList := setNodeList(n, f, nodeType)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_4NODEs_BYZANTINE_STUPID(t *testing.T) {
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Normal, Byzantine_Stupid}
	nodeList := setNodeList(n, f, nodeType)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	// wait for late messages
	time.Sleep(1 * time.Second)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_4NODEs_BYZANTINE_INTERCEPTOR(t *testing.T) {
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Normal, Byzantine_Interceptor}
	nodeList := setNodeList(n, f, nodeType)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	// wait for late messages
	time.Sleep(1 * time.Second)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_4NODEs_BYZANTINE_FAULT(t *testing.T) {
	// arbitrary value for network timeout to identify byzantine scenario
	const TIMEOUT = 3 * time.Second
	n := 4
	f := 1

	nodeType := []NodeType{Normal, Normal, Byzantine_Stupid, Byzantine_Stupid}
	nodeList := setNodeList(n, f, nodeType)
	doneChan := make(chan bool, len(nodeList))
	for i := 0; i < n; i++ {
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
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
	time.Sleep(1 * time.Second)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_16NODEs(t *testing.T) {
	n := 16
	f := 5

	nodeType := []NodeType{
		Normal, Normal, Normal, Normal,
		Normal, Normal, Normal, Normal,
		Normal, Normal, Normal, Normal,
		Normal, Normal, Normal, Normal,
	}
	nodeList := setNodeList(n, f, nodeType)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	// wait for late messages
	time.Sleep(1 * time.Second)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_16NODEs_BYZANTINE_MIX(t *testing.T) {
	n := 16
	f := 5

	nodeType := []NodeType{
		Normal, Normal, Normal, Normal,
		Normal, Normal, Normal, Normal,
		Normal, Normal, Normal,
		Byzantine_Stupid, Byzantine_Stupid, Byzantine_Stupid,
		Byzantine_Interceptor, Byzantine_Interceptor,
	}
	nodeList := setNodeList(n, f, nodeType)
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	// wait for late messages
	time.Sleep(1 * time.Second)
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_16NODEs_BYZANTINE_FAULT(t *testing.T) {
	// arbitrary value for network timeout to identify byzantine scenario
	const TIMEOUT = 3 * time.Second
	n := 16
	f := 5

	nodeType := []NodeType{
		Normal, Normal, Normal, Normal,
		Normal, Normal, Normal, Normal,
		Normal, Normal,
		Byzantine_Stupid, Byzantine_Stupid, Byzantine_Stupid,
		Byzantine_Stupid, Byzantine_Stupid, Byzantine_Stupid,
	}
	nodeList := setNodeList(n, f, nodeType)
	doneChan := make(chan bool, len(nodeList))
	for i := 0; i < n; i++ {
		go func(idx int) {
			data := fmt.Sprintf("node %d propose consensus", idx)
			if err := RBCTest(nodeList[idx], nodeList, []byte(data)); err != nil {
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
	time.Sleep(1 * time.Second)
	for _, node := range nodeList {
		node.Close()
	}
}
