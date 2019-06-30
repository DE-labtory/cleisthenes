package core

import (
	"bytes"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/honeybadger"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/tpke"
)

type sampleTransaction struct {
	From   string
	To     string
	Amount int
}

type mockACS struct {
	HandleInputFunc   func([]byte) error
	HandleMessageFunc func(member cleisthenes.Member, msg *pb.Message) error
}

func (a *mockACS) HandleInput(data []byte) error {
	return a.HandleInputFunc(data)
}
func (a *mockACS) HandleMessage(sender cleisthenes.Member, msg *pb.Message) error {
	return a.HandleMessageFunc(sender, msg)
}

//
type mockACSFactory struct {
	HandleInputFunc   func([]byte) error
	HandleMessageFunc func(member cleisthenes.Member, msg *pb.Message) error
}

func (f *mockACSFactory) Create() (honeybadger.ACS, error) {
	return &mockACS{
		HandleInputFunc:   f.HandleInputFunc,
		HandleMessageFunc: f.HandleMessageFunc,
	}, nil
}

func newNodeForTest(
	batchSize int,
	proposalInterval time.Duration,
	addrStr string,
	memberAddrList []string,
	acsFactory honeybadger.ACSFactory,
	txValidator cleisthenes.TxValidator,
	batchReceiver cleisthenes.BatchReceiver,
	resultSender cleisthenes.ResultSender,
) (Hbbft, error) {
	connPool := cleisthenes.NewConnectionPool()

	addr, err := cleisthenes.ToAddress(addrStr)
	if err != nil {
		return nil, err
	}

	memberMap := cleisthenes.NewMemberMap()
	memberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	for _, addrStr := range memberAddrList {
		addr, err := cleisthenes.ToAddress(addrStr)
		if err != nil {
			return nil, err
		}
		memberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	}

	txQueue := cleisthenes.NewTxQueue()
	hb := honeybadger.New(
		memberMap,
		acsFactory,
		&tpke.MockTpke{},
		batchReceiver,
		resultSender,
	)

	return &Node{
		addr: addr,
		txQueueManager: cleisthenes.NewDefaultTxQueueManager(
			txQueue,
			hb,
			batchSize,
			batchSize*len(memberMap.Members()),
			proposalInterval,
			txValidator,
		),
		resultReceiver:  cleisthenes.NewResultChannel(len(memberMap.Members())),
		messageEndpoint: hb,
		server:          cleisthenes.NewServer(addr),
		client:          cleisthenes.NewClient(),
		connPool:        connPool,
		memberMap:       memberMap,
	}, nil
}

func TestNodeWithMockACSAndMockTPKE(t *testing.T) {
	batchSize := 3
	proposalInterval := 1 * time.Second

	batchChan := cleisthenes.NewBatchChannel(10)

	// dataACSReceived saves data ACS received, because when honeybadger creates contribution
	// it random select tx from queue. so we cannot expect what order the transaction list will have.
	var dataACSReceived []byte

	acsFactory := &mockACSFactory{}
	acsFactory.HandleMessageFunc = func(member cleisthenes.Member, msg *pb.Message) error {
		return nil
	}
	acsFactory.HandleInputFunc = func(data []byte) error {
		dataACSReceived = data
		// assuming consensus time
		time.Sleep(200)
		batchChan.Send(cleisthenes.BatchMessage{
			Epoch: 0,
			Batch: map[cleisthenes.Member][]byte{
				cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5555}}: data,
				cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5556}}: data,
				cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5557}}: data,
				cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5558}}: data,
			},
		})
		return nil
	}

	txValidator := func(tx cleisthenes.Transaction) bool {
		return true
	}

	resultChan := cleisthenes.NewResultChannel(1)

	n, err := newNodeForTest(
		batchSize,
		proposalInterval,
		"localhost:5555",
		[]string{
			"localhost:5556",
			"localhost:5557",
			"localhost:5558",
		},
		acsFactory,
		txValidator,
		batchChan,
		resultChan,
	)
	if err != nil {
		t.Fatalf("failed to create node with err: %s", err.Error())
	}

	go n.Run()
	time.Sleep(500)

	n.Submit(sampleTransaction{From: "a", To: "b", Amount: 1})
	n.Submit(sampleTransaction{From: "b", To: "c", Amount: 1})
	n.Submit(sampleTransaction{From: "c", To: "d", Amount: 1})

	result := <-resultChan.Receive()
	if !bytes.Equal(
		result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5555}}], dataACSReceived) {
		t.Fatalf("expected result batch is %v, \n but got %v", dataACSReceived,
			result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5555}}])
	}
	if !bytes.Equal(
		result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5556}}], dataACSReceived) {
		t.Fatalf("expected result batch is %v, \n but got %v", dataACSReceived,
			result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5556}}])
	}
	if !bytes.Equal(
		result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5557}}], dataACSReceived) {
		t.Fatalf("expected result batch is %v, \n but got %v", dataACSReceived,
			result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5557}}])
	}
	if !bytes.Equal(
		result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5558}}], dataACSReceived) {
		t.Fatalf("expected result batch is %v, \n but got %v", dataACSReceived,
			result.Batch[cleisthenes.Member{Address: cleisthenes.Address{"localhost", 5558}}])
	}

}
