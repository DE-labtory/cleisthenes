package bba

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/test/mock"
)

type handleBvalTester struct {
	bvalList  []*bvalRequest
	assertMap map[int]func(t *testing.T, bbaInstance *BBA)
}

func newHandleBvalTester(bvalList []*bvalRequest) *handleBvalTester {
	return &handleBvalTester{
		bvalList:  bvalList,
		assertMap: make(map[int]func(t *testing.T, bbaInstance *BBA)),
	}
}

func (h *handleBvalTester) setupRequestList(bvalList []*bvalRequest) ([]request, error) {
	result := make([]request, 0)
	for i, bval := range bvalList {
		d, err := json.Marshal(bval)
		if err != nil {
			return nil, err
		}
		result = append(result, request{
			sender: cleisthenes.Member{Address: cleisthenes.Address{Ip: "localhost" + strconv.Itoa(i), Port: 8080}},
			data:   &pb.Message_Bba{Bba: &pb.BBA{Payload: d}},
		})
	}
	return result, nil
}

func (h *handleBvalTester) setupBvalRepository() cleisthenes.RequestRepository {
	bvalRepo := &mock.RequestRepository{
		ReqMap: make(map[cleisthenes.Address]cleisthenes.Request),
	}
	bvalRepo.SaveFunc = func(addr cleisthenes.Address, req cleisthenes.Request) error {
		bvalRepo.ReqMap[addr] = req
		return nil
	}
	bvalRepo.FindFunc = func(addr cleisthenes.Address) (cleisthenes.Request, error) {
		return bvalRepo.ReqMap[addr], nil
	}
	bvalRepo.FindAllFunc = func() []cleisthenes.Request {
		result := make([]cleisthenes.Request, 0)
		for _, req := range bvalRepo.ReqMap {
			result = append(result, req)
		}
		return result
	}
	return bvalRepo
}

func (h *handleBvalTester) setupBroadcaster(reqList []request) *mock.Broadcaster {
	broadcaster := &mock.Broadcaster{
		ConnMap:                make(map[cleisthenes.ConnId]mock.Connection),
		BroadcastedMessageList: make([]pb.Message, 0),
	}
	for _, req := range reqList {
		conn := mock.Connection{
			ConnId: req.sender.Address.String(),
		}
		conn.SendFunc = func(msg pb.Message, successCallBack func(interface{}), errCallBack func(error)) {
			broadcaster.BroadcastedMessageList = append(broadcaster.BroadcastedMessageList, msg)
		}
		broadcaster.ConnMap[req.sender.Address.String()] = conn
	}
	return broadcaster
}

func (h *handleBvalTester) setupAssert(i int, assert func(t *testing.T, bbaInstance *BBA)) {
	h.assertMap[i] = assert
}

func (h *handleBvalTester) assert(i int) (func(t *testing.T, bbaInstance *BBA), bool) {
	f, ok := h.assertMap[i]
	return f, ok
}

func handleBvalRequestTestSetup(t *testing.T, bvalList []*bvalRequest) (*BBA, *mock.Broadcaster, *handleBvalTester, func()) {
	tester := newHandleBvalTester(bvalList)
	reqList, err := tester.setupRequestList(bvalList)
	if err != nil {
		t.Fatalf("failed to setup request list: err=%s", err)
	}
	bvalRepo := tester.setupBvalRepository()
	broadcaster := tester.setupBroadcaster(reqList)

	bbaInstance := &BBA{
		n:                  10,
		f:                  3,
		bvalRepo:           bvalRepo,
		binValueChan:       make(chan struct{}),
		binValueSet:        newBinarySet(),
		broadcastedBvalSet: newBinarySet(),
		broadcaster:        broadcaster,
	}
	return bbaInstance, broadcaster, tester, func() {
		close(bbaInstance.binValueChan)
	}
}

func TestBBA_HandleBvalRequest(t *testing.T) {
	bvalList := []*bvalRequest{
		{Value: one},
		{Value: one},
		{Value: one},
		{Value: one},
		{Value: one},
		{Value: one},
		{Value: one},
	}

	bbaInstance, broadcaster, tester, teardown := handleBvalRequestTestSetup(t, bvalList)
	go func() {
		<-bbaInstance.binValueChan
		teardown()
	}()

	tester.setupAssert(3, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.broadcastedBvalSet.exist(one) {
			t.Fatalf("broadcasted bval set not include f+1 sent binary value")
		}

		//
		// test broadcasted message
		//
		if len(broadcaster.BroadcastedMessageList) != len(bvalList) {
			t.Fatalf("expected broadcasted message size is %d, but got %d", len(bvalList), len(broadcaster.BroadcastedMessageList))
		}
		for _, msg := range broadcaster.BroadcastedMessageList {
			req, ok := msg.Payload.(*pb.Message_Bba)
			if !ok {
				t.Fatalf("expected payload type is %+v, but got %+v", pb.Message_Bba{}, req)
			}
			receivedBval := &bvalRequest{}
			if err := json.Unmarshal(req.Bba.Payload, receivedBval); err != nil {
				t.Fatalf("unmarshal bval request failed with error: %s", err.Error())
			}
			if receivedBval.Value != one {
				t.Fatalf("expected bval is %t, but got %t", receivedBval.Value, one)
			}
		}
	})
	tester.setupAssert(6, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.binValueSet.exist(one) {
			t.Fatalf("binValue set not include f+1 sent binary value")
		}
	})

	for i, bval := range bvalList {
		sender := cleisthenes.Member{
			Address: cleisthenes.Address{
				Ip:   "localhost" + strconv.Itoa(i),
				Port: 8080,
			},
		}
		if err := bbaInstance.handleBvalRequest(sender, bval); err != nil {
			t.Fatalf("handle bval request failed with error: %s", err.Error())
		}

		if f, ok := tester.assert(i); ok {
			f(t, bbaInstance)
		}
	}
}

func TestBBA_HandleBvalRequest_OneZeroCombined(t *testing.T) {
	bvalList := []*bvalRequest{
		{Value: one},  // 0. (one, zero) = (1, 0)
		{Value: zero}, // 1. (one, zero) = (1, 1)
		{Value: zero}, // 2. (one, zero) = (1, 2)
		{Value: one},  // 3. (one, zero) = (2, 2)
		{Value: one},  // 4. (one, zero) = (3, 2)
		{Value: one},  // 5. (one, zero) = (4, 2), one broadcasted
		{Value: zero}, // 6. (one, zero) = (4, 3)
		{Value: zero}, // 7. (one, zero) = (4, 4), zero broadcasted
		{Value: one},  // 8. (one, zero)  = (5, 4)
		{Value: one},  // 9. (one, zero) = (6, 4)
	}

	bbaInstance, broadcaster, tester, teardown := handleBvalRequestTestSetup(t, bvalList)
	done := make(chan struct{})

	go func() {
		select {
		case <-bbaInstance.binValueChan:
			t.Fatalf("binValue should not be set")
		case <-done:
		}

		teardown()
	}()

	tester.setupAssert(5, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.broadcastedBvalSet.exist(one) {
			t.Fatalf("broadcasted bval set not include f+1 sent binary value")
		}

		//
		// test broadcasted message
		//
		if len(broadcaster.BroadcastedMessageList) != len(bvalList) {
			t.Fatalf("expected broadcasted message size is %d, but got %d", len(bvalList), len(broadcaster.BroadcastedMessageList))
		}
		for _, msg := range broadcaster.BroadcastedMessageList {
			req, ok := msg.Payload.(*pb.Message_Bba)
			if !ok {
				t.Fatalf("expected payload type is %+v, but got %+v", pb.Message_Bba{}, req)
			}
			receivedBval := &bvalRequest{}
			if err := json.Unmarshal(req.Bba.Payload, receivedBval); err != nil {
				t.Fatalf("unmarshal bval request failed with error: %s", err.Error())
			}
			if receivedBval.Value != one {
				t.Fatalf("expected bval is %t, but got %t", receivedBval.Value, one)
			}
		}
	})
	tester.setupAssert(7, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.broadcastedBvalSet.exist(zero) {
			t.Fatalf("broadcasted bval set not include f+1 sent binary value")
		}
		//
		// test broadcasted message
		//
		if len(broadcaster.BroadcastedMessageList) != 2*len(bvalList) {
			t.Fatalf("expected broadcasted message size is %d, but got %d", 2*len(bvalList), len(broadcaster.BroadcastedMessageList))
		}
		for _, msg := range broadcaster.BroadcastedMessageList[len(bvalList):] {
			req, ok := msg.Payload.(*pb.Message_Bba)
			if !ok {
				t.Fatalf("expected payload type is %+v, but got %+v", pb.Message_Bba{}, req)
			}
			receivedBval := &bvalRequest{}
			if err := json.Unmarshal(req.Bba.Payload, receivedBval); err != nil {
				t.Fatalf("unmarshal bval request failed with error: %s", err.Error())
			}
			if receivedBval.Value != zero {
				t.Fatalf("expected bval is %t, but got %t", receivedBval.Value, zero)
			}
		}
	})
	tester.setupAssert(9, func(t *testing.T, bbaInstance *BBA) {
		if bbaInstance.binValueSet.exist(one) {
			t.Fatalf("binValue set **include** f+1 sent binary value")
		}
		if bbaInstance.binValueSet.exist(zero) {
			t.Fatalf("binValue set **include** f+1 sent binary value")
		}
	})

	for i, bval := range bvalList {
		sender := cleisthenes.Member{
			Address: cleisthenes.Address{
				Ip:   "localhost" + strconv.Itoa(i),
				Port: 8080,
			},
		}
		if err := bbaInstance.handleBvalRequest(sender, bval); err != nil {
			t.Fatalf("handle bval request failed with error: %s", err.Error())
		}

		if f, ok := tester.assert(i); ok {
			f(t, bbaInstance)
		}
	}
	done <- struct{}{}
}
