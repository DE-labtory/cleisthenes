package integration

import (
	"log"
	"net/http"
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

func setNodeList(n, f int, txValidator cleisthenes.TxValidator, nodeType []NodeType) []*Node {
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

		// messageEndpoint
		mNode.messageEndpoint = hb

		// batchSize = λN^2*logN
		//batchSize := n*n*int(math.Log2(float64(n)))
		batchSize := n // for test
		txQueueManager := cleisthenes.NewDefaultTxQueueManager(
			cleisthenes.NewTxQueue(),
			hb,
			batchSize/n,
			batchSize,
			time.Duration(500),
			txValidator,
		)

		mNode.txQueueManager = txQueueManager
	}

	return nodeList
}

func IntegrationTest(proposer *Node, transactionCount int) error {
	for idx := 0; idx < transactionCount; idx++ {
		proposer.AddTransaction(exTransaction{Amount: idx})
	}

	for {
		select {
		case <-proposer.doneChan:
			if transactionCount == proposer.resultBatch.len() {
				return nil
			}
		}
	}
}

func Test_TX_4NODEs(t *testing.T) {
	n := 4
	f := 1
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	txValidator := func(tx cleisthenes.Transaction) bool {
		return true
	}

	nodeType := []NodeType{Normal, Normal, Normal, Normal}
	nodeList := setNodeList(n, f, txValidator, nodeType)
	transactionCount := 10
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			if err := IntegrationTest(nodeList[idx], transactionCount); err != nil {
				t.Fatalf("test fail : %s", err.Error())
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	for _, node := range nodeList {
		if node.resultBatch.len() != transactionCount {
			t.Fatalf("expected transaction count : %d, got : %d", transactionCount, node.resultBatch.len())
		}
	}

	// wait for late messages
	time.Sleep(1000 * time.Millisecond)
	for _, node := range nodeList {
		node.Close()
	}
}
