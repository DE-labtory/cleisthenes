package rbc

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes/log"

	"github.com/DE-labtory/iLogger"

	"github.com/DE-labtory/cleisthenes/test/util"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/rbc"
)

func setNodeList(n, f int, nodeType []NodeType) []*Node {
	var ip string = "127.0.0.1"
	var port uint16 = 8000

	nodeList := make([]*Node, 0)

	for i := 0; i < n; i++ {
		availablePort := util.GetAvailablePort(port)
		address := cleisthenes.Address{
			Ip:   ip,
			Port: availablePort,
		}
		node := NewNode(n, f, address, nodeType[i])
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

		owner := mNode.memberMap.Member(mNode.address)
		for _, member := range mNode.memberMap.Members() {
			r, _ := rbc.New(n, f, owner, member, mNode.connPool)
			mNode.rbcMap[member.Address] = r
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

	// polling value
	doneCnt := 0
	for {
		for _, node := range nodeList {
			if value := node.Value(proposer.address); value != nil {
				if !bytes.Equal(value, data) {
					return errors.New(fmt.Sprintf("node : %s fail to reach an agreement - expected : %s, got : %s\n", node.address.String(), data, value))
				} else {
					log.Info("action", "start", "owner", node.address.String())
				}
				doneCnt++
				break
			}
		}

		if doneCnt == nodeCnt-byzantineCnt {
			break
		}
	}

	return nil
}

func Test_RBC_4NODEs(t *testing.T) {
	n := 4
	f := 1

	iLogger.SetToDebug()
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
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_4NODEs_BYZANTINE_FAULT(t *testing.T) {
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
	for _, node := range nodeList {
		node.Close()
	}
}

func Test_RBC_16NODEs_BYZANTINE_FAULT(t *testing.T) {
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

	for _, node := range nodeList {
		node.Close()
	}
}
