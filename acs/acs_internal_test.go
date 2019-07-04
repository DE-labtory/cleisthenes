package acs

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/bba"
	"github.com/DE-labtory/cleisthenes/pb"
)

type mockRBC struct {
	owner           *cleisthenes.Member
	expected        []byte
	HandleInputFunc func(data []byte) error
}

func (rbc *mockRBC) HandleInput(data []byte) error {
	return rbc.HandleInputFunc(data)
}

func (rbc *mockRBC) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	return nil
}

func (rbc *mockRBC) Close() {}

type mockBBA struct {
	owner           *cleisthenes.Member
	HandleInputFunc func(bvalRequest *bba.BvalRequest) error
}

func (bba *mockBBA) HandleInput(bvalRequest *bba.BvalRequest) error {
	return bba.HandleInputFunc(bvalRequest)
}

func (bba *mockBBA) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	return nil
}

func (bba *mockBBA) Idle() bool {
	return true
}

func (bba *mockBBA) Close() {}

func setupACS(t *testing.T, n, f int, owner cleisthenes.Member, memberList []cleisthenes.Member) *ACS {
	rbcRepo := NewRBCRepository()
	bbaRepo := NewBBARepository()
	broadcastResult := NewbroadcastDataMap()
	agreementResult := NewBinaryStateMap()
	agreementStarted := NewBinaryStateMap()
	for _, member := range memberList {
		r := &mockRBC{owner: &member}
		b := &mockBBA{owner: &member}
		b.HandleInputFunc = func(bvalRequest *bba.BvalRequest) error {
			if bvalRequest.Value != cleisthenes.One {
				t.Fatalf("invalid bvalRequest - expected : %t, got : %t", cleisthenes.One, bvalRequest.Value)
			}

			return nil
		}
		rbcRepo.Save(member, r)
		bbaRepo.Save(member, b)
		broadcastResult.set(member, []byte{})
		agreementResult.set(member, cleisthenes.BinaryState{})
		agreementStarted.set(member, cleisthenes.BinaryState{})
	}

	return &ACS{
		n:                n,
		f:                f,
		owner:            owner,
		rbcRepo:          rbcRepo,
		bbaRepo:          bbaRepo,
		broadcastResult:  broadcastResult,
		agreementResult:  agreementResult,
		agreementStarted: agreementStarted,
		reqChan:          make(chan request),
		agreementChan:    make(chan struct{}, n),
		closeChan:        make(chan struct{}, 1),
		dataReceiver:     cleisthenes.NewDataChannel(n),
		binaryReceiver:   cleisthenes.NewBinaryChannel(n),
		batchSender:      cleisthenes.NewBatchChannel(),
	}
}

func TestACS_sendZeroToIdleBba(t *testing.T) {
	n := 4
	f := 1

	memberList := make([]cleisthenes.Member, 0)
	for idx := 0; idx < n; idx++ {
		member := cleisthenes.NewMember("127.0.0.1", uint16(idx))
		memberList = append(memberList, *member)
	}

	owner := memberList[0]
	acs := setupACS(t, n, f, owner, memberList)

	acs.sendZeroToIdleBba()

	for _, state := range acs.agreementStarted.itemMap() {
		if state.Value() || state.Undefined() {
			t.Fatalf("invalid state - expected value : %t, got : %t, expected undefined : %t, got : %t", cleisthenes.Zero, state.Value(), cleisthenes.Zero, state.Undefined())
		}
	}
}

func TestACS_tryCompleteAgreement(t *testing.T) {
	n := 4
	f := 1

	memberList := make([]cleisthenes.Member, 0)
	for idx := 0; idx < n; idx++ {
		member := cleisthenes.NewMember("127.0.0.1", uint16(idx))
		memberList = append(memberList, *member)
	}

	owner := memberList[0]
	acs := setupACS(t, n, f, owner, memberList)

	for idx := 0; idx < n-f; idx++ {
		trueState := cleisthenes.BinaryState{}
		trueState.Set(true)
		acs.agreementResult.set(memberList[idx], trueState)
	}

	for idx := 0; idx < n-f-1; idx++ {
		data := []byte(strconv.Itoa(idx))
		acs.broadcastResult.set(memberList[idx], data)
	}

	acs.tryCompleteAgreement()

	if acs.dec.Value() {
		t.Fatal("error : broadcast is not over yet")
	}

	acs.broadcastResult.set(memberList[n-f-1], []byte(strconv.Itoa(n-f-1)))
	acs.tryCompleteAgreement()
	if !acs.dec.Value() {
		t.Fatal("error : broadcast is over already")
	}
}

func TestACS_tryAgreementStart(t *testing.T) {
	n := 4
	f := 1

	memberList := make([]cleisthenes.Member, 0)
	for idx := 0; idx < n; idx++ {
		member := cleisthenes.NewMember("127.0.0.1", uint16(idx))
		memberList = append(memberList, *member)
	}

	owner := memberList[0]
	acs := setupACS(t, n, f, owner, memberList)

	for _, member := range memberList {
		acs.tryAgreementStart(member)
	}

	for _, member := range memberList {
		if state := acs.agreementStarted.item(member); !state.Value() {
			t.Fatalf("invalid state - expected : %t, got : %t", true, state.Value())
		}
	}
}

func TestACS_processData(t *testing.T) {
	n := 4
	f := 1

	memberList := make([]cleisthenes.Member, 0)
	for idx := 0; idx < n; idx++ {
		member := cleisthenes.NewMember("127.0.0.1", uint16(idx))
		memberList = append(memberList, *member)
	}

	owner := memberList[0]
	acs := setupACS(t, n, f, owner, memberList)

	for idx, member := range memberList {
		data := []byte(strconv.Itoa(idx))
		acs.processData(member, data)
	}

	for idx, member := range memberList {
		if data := acs.broadcastResult.item(member); !bytes.Equal(data, []byte(strconv.Itoa(idx))) {
			t.Fatalf("invalid data - expected : %s, got : %s", []byte(strconv.Itoa(idx)), data)
		}
	}
}

func TestACS_processAgreement(t *testing.T) {
	n := 4
	f := 1

	memberList := make([]cleisthenes.Member, 0)
	for idx := 0; idx < n; idx++ {
		member := cleisthenes.NewMember("127.0.0.1", uint16(idx))
		memberList = append(memberList, *member)
	}

	owner := memberList[0]
	acs := setupACS(t, n, f, owner, memberList)

	for _, member := range memberList {
		acs.processAgreement(member, true)
	}

	for _, member := range memberList {
		if state := acs.agreementResult.item(member); !state.Value() {
			t.Fatalf("invalid binary - expected : %t, got : %t", true, state.Value())
		}
	}
}
