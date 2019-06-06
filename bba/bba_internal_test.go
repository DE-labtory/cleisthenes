package bba

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/test/mock"
)

var defaultTestTimeout = 3 * time.Second

type bbaTester struct {
	timer     *time.Timer
	assertMap map[int]func(t *testing.T, bbaInstance *BBA)
}

func newBbaTester() *bbaTester {
	return &bbaTester{
		timer:     time.NewTimer(defaultTestTimeout),
		assertMap: make(map[int]func(t *testing.T, bbaInstance *BBA)),
	}
}

func (h *bbaTester) waitWithTimer(done chan struct{}) error {
	select {
	case <-h.timer.C:
		return errors.New("test failed with timeout")
	case <-done:
		return nil
	}
}

func (h *bbaTester) setupBvalRequestList(bvalList []*bvalRequest) ([]request, error) {
	result := make([]request, 0)
	for i, bval := range bvalList {
		d, err := json.Marshal(bval)
		if err != nil {
			return nil, err
		}
		result = append(result, request{
			sender: cleisthenes.Member{Address: cleisthenes.Address{Ip: "localhost" + strconv.Itoa(i), Port: 8080}},
			data: &pb.Message_Bba{
				Bba: &pb.BBA{
					Type:    pb.BBA_BVAL,
					Payload: d,
				},
			},
		})
	}
	return result, nil
}

func (h *bbaTester) setupAuxRequestList(bvalList []*auxRequest) ([]request, error) {
	result := make([]request, 0)
	for i, bval := range bvalList {
		d, err := json.Marshal(bval)
		if err != nil {
			return nil, err
		}
		result = append(result, request{
			sender: cleisthenes.Member{Address: cleisthenes.Address{Ip: "localhost" + strconv.Itoa(i), Port: 8080}},
			data: &pb.Message_Bba{
				Bba: &pb.BBA{
					Type:    pb.BBA_AUX,
					Payload: d,
				},
			},
		})
	}
	return result, nil
}

func (h *bbaTester) setupRequestRepository() cleisthenes.RequestRepository {
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

func (h *bbaTester) setupBroadcaster(reqList []request) *mock.Broadcaster {
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

func (h *bbaTester) setupAssert(i int, assert func(t *testing.T, bbaInstance *BBA)) {
	h.assertMap[i] = assert
}

func (h *bbaTester) assert(i int) (func(t *testing.T, bbaInstance *BBA), bool) {
	f, ok := h.assertMap[i]
	return f, ok
}

func handleBvalRequestTestSetup(t *testing.T, bvalList []*bvalRequest) (*BBA, *mock.Broadcaster, *bbaTester, func()) {
	tester := newBbaTester()
	reqList, err := tester.setupBvalRequestList(bvalList)
	if err != nil {
		t.Fatalf("failed to setup request list: err=%s", err)
	}
	bvalRepo := tester.setupRequestRepository()
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
	done := make(chan struct{})
	bvalList := []*bvalRequest{
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
	}

	bbaInstance, broadcaster, tester, teardown := handleBvalRequestTestSetup(t, bvalList)
	go func() {
		<-bbaInstance.binValueChan
		teardown()
		done <- struct{}{}
	}()

	tester.setupAssert(3, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.broadcastedBvalSet.exist(cleisthenes.One) {
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
			if receivedBval.Value != cleisthenes.One {
				t.Fatalf("expected bval is %t, but got %t", receivedBval.Value, cleisthenes.One)
			}
		}
	})
	tester.setupAssert(6, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.binValueSet.exist(cleisthenes.One) {
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
	if err := tester.waitWithTimer(done); err != nil {
		t.Fatal(err)
	}
}

func TestBBA_HandleBvalRequest_OneZeroCombined(t *testing.T) {
	bvalList := []*bvalRequest{
		{Value: cleisthenes.One},  // 0. (one, zero) = (1, 0)
		{Value: cleisthenes.Zero}, // 1. (one, zero) = (1, 1)
		{Value: cleisthenes.Zero}, // 2. (one, zero) = (1, 2)
		{Value: cleisthenes.One},  // 3. (one, zero) = (2, 2)
		{Value: cleisthenes.One},  // 4. (one, zero) = (3, 2)
		{Value: cleisthenes.One},  // 5. (one, zero) = (4, 2), one broadcasted
		{Value: cleisthenes.Zero}, // 6. (one, zero) = (4, 3)
		{Value: cleisthenes.Zero}, // 7. (one, zero) = (4, 4), zero broadcasted
		{Value: cleisthenes.One},  // 8. (one, zero)  = (5, 4)
		{Value: cleisthenes.One},  // 9. (one, zero) = (6, 4)
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
		if !bbaInstance.broadcastedBvalSet.exist(cleisthenes.One) {
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
			if receivedBval.Value != cleisthenes.One {
				t.Fatalf("expected bval is %t, but got %t", receivedBval.Value, cleisthenes.One)
			}
		}
	})
	tester.setupAssert(7, func(t *testing.T, bbaInstance *BBA) {
		if !bbaInstance.broadcastedBvalSet.exist(cleisthenes.Zero) {
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
			if receivedBval.Value != cleisthenes.Zero {
				t.Fatalf("expected bval is %t, but got %t", receivedBval.Value, cleisthenes.Zero)
			}
		}
	})
	tester.setupAssert(9, func(t *testing.T, bbaInstance *BBA) {
		if bbaInstance.binValueSet.exist(cleisthenes.One) {
			t.Fatalf("binValue set **include** f+1 sent binary value")
		}
		if bbaInstance.binValueSet.exist(cleisthenes.Zero) {
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

func handleAuxRequestTestSetup(t *testing.T, auxList []*auxRequest) (*BBA, *mock.Broadcaster, *bbaTester, func()) {
	tester := newBbaTester()
	reqList, err := tester.setupAuxRequestList(auxList)
	if err != nil {
		t.Fatalf("failed to setup request list: err=%s", err)
	}
	bvalRepo := tester.setupRequestRepository()
	auxRepo := tester.setupRequestRepository()
	broadcaster := tester.setupBroadcaster(reqList)

	bbaInstance := &BBA{
		n:                   10,
		f:                   3,
		bvalRepo:            bvalRepo,
		auxRepo:             auxRepo,
		tryoutAgreementChan: make(chan struct{}, 1),
		binValueSet:         newBinarySet(),
		broadcastedBvalSet:  newBinarySet(),
		broadcaster:         broadcaster,
	}
	return bbaInstance, broadcaster, tester, func() {
		close(bbaInstance.tryoutAgreementChan)
	}
}

func TestBBA_HandleAuxRequest(t *testing.T) {
	done := make(chan struct{})

	auxList := []*auxRequest{
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
		{Value: cleisthenes.One},
	}

	bbaInstance, _, tester, teardown := handleAuxRequestTestSetup(t, auxList)
	defer teardown()

	go func() {
		<-bbaInstance.tryoutAgreementChan
		t.Log("tryoutAgreementChan got signal")
		done <- struct{}{}
	}()

	tester.setupAssert(6, func(t *testing.T, bbaInstance *BBA) {
		reqList := bbaInstance.auxRepo.FindAll()
		if len(reqList) != 7 {
			t.Fatalf("expected request list length is %d, but got %d", 7, len(reqList))
		}
		for i, req := range reqList {
			aux, ok := req.(*auxRequest)
			if !ok {
				t.Fatalf("request is not auxRequest type: [%d]", i)
			}
			if aux.Value != cleisthenes.One {
				t.Fatalf("request value is not one")
			}
		}
	})

	for i, aux := range auxList {
		sender := cleisthenes.Member{
			Address: cleisthenes.Address{
				Ip:   "localhost" + strconv.Itoa(i),
				Port: 8080,
			},
		}
		if err := bbaInstance.handleAuxRequest(sender, aux); err != nil {
			t.Fatalf("handle bval request failed with error: %s", err.Error())
		}

		if f, ok := tester.assert(i); ok {
			f(t, bbaInstance)
		}
	}
	if err := tester.waitWithTimer(done); err != nil {
		t.Fatal(err)
	}
}

func TestBBA_HandleAuxRequest_OneZeroCombined(t *testing.T) {
	done := make(chan struct{})

	auxList := []*auxRequest{
		{Value: cleisthenes.One},  // 0. (one, zero) = (1, 0)
		{Value: cleisthenes.Zero}, // 1. (one, zero) = (1, 1)
		{Value: cleisthenes.Zero}, // 2. (one, zero) = (1, 2)
		{Value: cleisthenes.One},  // 3. (one, zero) = (2, 2)
		{Value: cleisthenes.One},  // 4. (one, zero) = (3, 2)
		{Value: cleisthenes.One},  // 5. (one, zero) = (4, 2), one broadcasted
		{Value: cleisthenes.Zero}, // 6. (one, zero) = (4, 3)
		{Value: cleisthenes.Zero}, // 7. (one, zero) = (4, 4), zero broadcasted
		{Value: cleisthenes.One},  // 8. (one, zero)  = (5, 4)
		{Value: cleisthenes.One},  // 9. (one, zero) = (6, 4)
		{Value: cleisthenes.One},  // 10. (one, zero) = (7, 4)
	}

	bbaInstance, _, tester, teardown := handleAuxRequestTestSetup(t, auxList)
	defer teardown()

	go func() {
		// should receive signal from tryoutAgreementChan
		<-bbaInstance.tryoutAgreementChan
		t.Log("tryoutAgreementChan got signal")
		done <- struct{}{}
	}()

	tester.setupAssert(10, func(t *testing.T, bbaInstance *BBA) {

		reqList := bbaInstance.auxRepo.FindAll()
		if len(reqList) != 11 {
			t.Fatalf("expected request list length is %d, but got %d", 11, len(reqList))
		}

		oneCount := 0
		for i, req := range reqList {
			aux, ok := req.(*auxRequest)
			if !ok {
				t.Fatalf("request is not auxRequest type: [%d]", i)
			}
			if aux.Value == cleisthenes.One {
				oneCount++
			}
		}
		if oneCount != 7 {
			t.Fatalf("expected number of one is %d, but got %d", 7, oneCount)
		}
	})

	for i, aux := range auxList {
		sender := cleisthenes.Member{
			Address: cleisthenes.Address{
				Ip:   "localhost" + strconv.Itoa(i),
				Port: 8080,
			},
		}
		if err := bbaInstance.handleAuxRequest(sender, aux); err != nil {
			t.Fatalf("handle bval request failed with error: %s", err.Error())
		}

		if f, ok := tester.assert(i); ok {
			f(t, bbaInstance)
		}
	}
	if err := tester.waitWithTimer(done); err != nil {
		t.Fatal(err)
	}
}

func tryoutAgreementTestSetup() (*BBA, *bbaTester, func()) {
	coinGenerator := &mock.CoinGenerator{}
	bbaInstance := &BBA{
		n:                  10,
		f:                  3,
		coinGenerator:      coinGenerator,
		advanceRoundChan:   make(chan struct{}, 1),
		dec:                cleisthenes.NewBinaryState(),
		est:                cleisthenes.NewBinaryState(),
		binValueSet:        newBinarySet(),
		broadcastedBvalSet: newBinarySet(),
	}
	return bbaInstance, newBbaTester(), func() {
		close(bbaInstance.advanceRoundChan)
	}
}

func TestBBA_TryoutAgreement_WhenBinValueSizeIsOne_AndSameWithCoinValue(t *testing.T) {
	done := make(chan struct{})
	bbaInstance, tester, teardown := tryoutAgreementTestSetup()
	defer teardown()

	// given
	bbaInstance.binValueSet.union(cleisthenes.One)

	go func() {
		<-bbaInstance.advanceRoundChan
		done <- struct{}{}
	}()

	bbaInstance.tryoutAgreement()

	if bbaInstance.dec.Undefined() {
		t.Fatal("expected decision value should not be undefined")
	}
	if !bbaInstance.done {
		t.Fatalf("bba is not done")
	}

	if err := tester.waitWithTimer(done); err != nil {
		t.Fatal(err)
	}
}

func TestBBA_TryoutAgreement_WhenBinValueSizeIsOne_AndDifferentWithCoinValue(t *testing.T) {
	done := make(chan struct{})
	bbaInstance, tester, teardown := tryoutAgreementTestSetup()
	defer teardown()

	// given
	bbaInstance.binValueSet.union(cleisthenes.Zero)

	go func() {
		<-bbaInstance.advanceRoundChan
		done <- struct{}{}
	}()

	bbaInstance.tryoutAgreement()

	if !bbaInstance.dec.Undefined() {
		t.Fatal("expected decision value should not be undefined")
	}
	if bbaInstance.est.Value() != cleisthenes.Zero {
		t.Fatalf("expected estimation value is %v, but got %v", cleisthenes.Zero, bbaInstance.est)
	}
	if bbaInstance.done {
		t.Fatalf("bba should not be done")
	}

	if err := tester.waitWithTimer(done); err != nil {
		t.Fatal(err)
	}
}

func TestBBA_TryoutAgreement_WhenBinValueSizeIsTwo(t *testing.T) {
	done := make(chan struct{})
	bbaInstance, tester, teardown := tryoutAgreementTestSetup()
	defer teardown()

	// given
	bbaInstance.binValueSet.union(cleisthenes.Zero)
	bbaInstance.binValueSet.union(cleisthenes.One)

	go func() {
		<-bbaInstance.advanceRoundChan
		done <- struct{}{}
	}()

	bbaInstance.tryoutAgreement()

	if !bbaInstance.dec.Undefined() {
		t.Fatal("expected decision value should be undefined")
	}
	if bbaInstance.est.Value() != cleisthenes.Binary(mock.Coin) {
		t.Fatalf("expected estimation value is %v, but got %v", cleisthenes.One, bbaInstance.est)
	}
	if bbaInstance.done {
		t.Fatalf("bba should not be done")
	}

	if err := tester.waitWithTimer(done); err != nil {
		t.Fatal(err)
	}
}
