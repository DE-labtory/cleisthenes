package honeybadger

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/acs"
	"github.com/DE-labtory/cleisthenes/honeybadger"
	"github.com/DE-labtory/cleisthenes/test/util"
	"github.com/DE-labtory/cleisthenes/tpke"
)

type mockACSFactory struct {
	n              int
	f              int
	acsOwner       cleisthenes.Member
	batchSender    cleisthenes.BatchSender
	memberMap      cleisthenes.MemberMap
	dataReceiver   cleisthenes.DataReceiver
	dataSender     cleisthenes.DataSender
	binaryReceiver cleisthenes.BinaryReceiver
	binarySender   cleisthenes.BinarySender
	broadcaster    cleisthenes.Broadcaster
}

func NewMockACSFactory(
	n int,
	f int,
	acsOwner cleisthenes.Member,
	memberMap cleisthenes.MemberMap,
	dataReceiver cleisthenes.DataReceiver,
	dataSender cleisthenes.DataSender,
	binaryReceiver cleisthenes.BinaryReceiver,
	binarySender cleisthenes.BinarySender,
	batchSender cleisthenes.BatchSender,
	broadcaster cleisthenes.Broadcaster,
) *mockACSFactory {
	return &mockACSFactory{
		n:              n,
		f:              f,
		acsOwner:       acsOwner,
		memberMap:      memberMap,
		dataReceiver:   dataReceiver,
		dataSender:     dataSender,
		binaryReceiver: binaryReceiver,
		binarySender:   binarySender,
		batchSender:    batchSender,
		broadcaster:    broadcaster,
	}
}

func (f *mockACSFactory) Create(epoch cleisthenes.Epoch) (honeybadger.ACS, error) {
	dataChan := cleisthenes.NewDataChannel(f.n)
	binaryChan := cleisthenes.NewBinaryChannel(f.n)

	return acs.New(
		f.n,
		f.f,
		epoch,
		f.acsOwner,
		f.memberMap,
		dataChan,
		dataChan,
		binaryChan,
		binaryChan,
		f.batchSender,
		f.broadcaster,
	)
}

func setNodeList(n, f int, nodeType []NodeType) []*Node {
	var ip string = "127.0.0.1"
	var port uint16 = 8000

	resultChanList := make([]*cleisthenes.ResultChannel, 0)
	nodeList := make([]*Node, 0)
	for i := 0; i < n; i++ {
		availablePort := util.GetAvailablePort(port)
		owner := cleisthenes.NewMember(ip, availablePort)
		resultChan := cleisthenes.NewResultChannel(1000)
		resultChanList = append(resultChanList, resultChan)
		node := NewNode(n, f, *owner, resultChan, nodeType[i])
		node.Run()
		time.Sleep(50 * time.Millisecond)
		nodeList = append(nodeList, node)

		port = availablePort + 1
	}

	for m, mNode := range nodeList {

		//myself
		member := cleisthenes.NewMember(mNode.owner.Address.Ip, mNode.owner.Address.Port)
		mNode.memberMap.Add(member)
		for y, yNode := range nodeList {
			if m != y {
				mNode.Connect(yNode.Address())
			}
		}

		batchChan := cleisthenes.NewBatchChannel(1000)
		dataChan := cleisthenes.NewDataChannel(n)
		binaryChan := cleisthenes.NewBinaryChannel(n)

		acsFactory := NewMockACSFactory(
			n,
			f,
			mNode.owner,
			*mNode.memberMap,
			dataChan,
			dataChan,
			binaryChan,
			binaryChan,
			batchChan,
			mNode.connPool,
		)

		hb := honeybadger.New(
			mNode.memberMap,
			acsFactory,
			&tpke.MockTpke{},
			batchChan,
			resultChanList[m],
		)

		mNode.hb = hb
	}

	return nodeList
}

func HBTest(proposer *Node, contribution cleisthenes.Contribution) error {
	if proposer.typ == Normal {
		proposer.Propose(contribution)
	}

	for {
		select {
		case <-proposer.doneChan:
			//fmt.Println("done")
			return nil
		}
	}
}

func Test_HB_4NODEs(t *testing.T) {
	n := 4
	f := 1
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	//runtime.GOMAXPROCS(runtime.NumCPU())

	nodeType := []NodeType{Normal, Normal, Normal, Normal}
	nodeList := setNodeList(n, f, nodeType)
	for cnt := 0; cnt < 100; cnt++ {
		wg := sync.WaitGroup{}
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				transaction := struct {
					From   string
					To     string
					Amount int
				}{
					From:   nodeList[idx].owner.Address.String(),
					To:     nodeList[idx].owner.Address.String(),
					Amount: idx,
				}

				if err := HBTest(nodeList[idx], cleisthenes.Contribution{
					TxList: []cleisthenes.Transaction{transaction},
				}); err != nil {
					t.Fatalf("test fail : %s", err.Error())
				}
				wg.Done()
			}(i)
		}

		wg.Wait()
		//fmt.Println("\033[2J")
		//os.RemoveAll("./log.log")
		//time.Sleep(10 * time.Millisecond)
	}

	for _, node := range nodeList {
		for epoch, result := range node.resultBatch {
			for member, batch := range result {
				fmt.Printf("node : %s, epoch : %d, member : %s, batch : %s\n", node.owner.Address.String(), epoch, member.Address.String(), batch)
			}
		}
	}

	// wait for late messages
	time.Sleep(100 * time.Millisecond)
	for _, node := range nodeList {
		node.Close()
	}
}
