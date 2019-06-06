package rbc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/rbc/merkletree"
	"github.com/DE-labtory/iLogger"
	"github.com/golang/protobuf/ptypes"
	"github.com/klauspost/reedsolomon"
)

var ErrInvalidRBCType = errors.New("invalid RBC message type")

type RBC struct {
	// number of network nodes
	n int

	// number of byzantine nodes which can tolerate
	f int

	// owner of rbc instance (node)
	owner cleisthenes.Member

	// proposerId is the ID of proposing node
	proposer cleisthenes.Member

	// Erasure coding using reed-solomon method
	enc reedsolomon.Encoder

	// output of RBC
	value []byte

	// length of original data
	contentLength uint64

	// number of sharded data and parity
	// data : N - F, parity : F
	numDataShards, numParityShards int

	// Request of other rbcs
	echoReqRepo  cleisthenes.RequestRepository
	readyReqRepo cleisthenes.RequestRepository

	valReceived, echoSent, readySent *cleisthenes.BinaryState

	// internal channels to communicate with other components
	closeChan chan struct{}
	reqChan   chan request
	lock      sync.RWMutex

	broadcaster cleisthenes.Broadcaster
}

func New(n, f int, owner, proposer cleisthenes.Member, broadcaster cleisthenes.Broadcaster) *RBC {
	numParityShards := 2 * f
	numDataShards := n - numParityShards

	enc, err := reedsolomon.New(numDataShards, numParityShards)
	if err != nil {
		panic(err)
	}

	echoReqRepo := NewEchoReqRepository()
	readyReqRepo := NewReadyReqRepository()
	rbc := &RBC{
		n:               n,
		f:               f,
		owner:           owner,
		proposer:        proposer,
		enc:             enc,
		numDataShards:   numDataShards,
		numParityShards: numParityShards,
		echoReqRepo:     echoReqRepo,
		readyReqRepo:    readyReqRepo,
		valReceived:     cleisthenes.NewBinaryState(),
		echoSent:        cleisthenes.NewBinaryState(),
		readySent:       cleisthenes.NewBinaryState(),
		closeChan:       make(chan struct{}),
		reqChan:         make(chan request),
		lock:            sync.RWMutex{},
		broadcaster:     broadcaster,
	}
	go rbc.run()
	return rbc
}

func (rbc *RBC) distributeMessage(proposer cleisthenes.Member, reqs []cleisthenes.Request) error {
	var typ pb.RBCType
	switch reqs[0].(type) {
	case *ValRequest:
		typ = pb.RBC_VAL
	default:
		return errors.New("invalid distributeMessage message type")
	}

	msgList := make([]pb.Message, 0)
	for _, req := range reqs {
		payload, err := json.Marshal(req)
		if err != nil {
			return err
		}

		msgList = append(msgList, pb.Message{
			Sender:    rbc.owner.Address.String(),
			Timestamp: ptypes.TimestampNow(),
			Payload: &pb.Message_Rbc{
				Rbc: &pb.RBC{
					Payload:       payload,
					Proposer:      proposer.Address.String(),
					ContentLength: rbc.contentLength,
					Type:          typ,
				},
			},
		})
	}
	rbc.broadcaster.DistributeMessage(msgList)

	return nil
}

func (rbc *RBC) shareMessage(proposer cleisthenes.Member, req cleisthenes.Request) error {
	var typ pb.RBCType
	switch req.(type) {
	case *EchoRequest:
		typ = pb.RBC_ECHO
	case *ReadyRequest:
		typ = pb.RBC_READY
	default:
		return errors.New("invalid shareMessage message type")
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	rbc.broadcaster.ShareMessage(pb.Message{
		Sender:    rbc.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload:       payload,
				Proposer:      proposer.Address.String(),
				ContentLength: rbc.contentLength,
				Type:          typ,
			},
		},
	})
	return nil
}

// MakeReqAndBroadcast make requests and broadcast to other nodes
// it is used in ACS
func (rbc *RBC) HandleInput(data []byte) error {
	shards, err := shard(rbc.enc, data)
	if err != nil {
		return err
	}

	reqs, err := makeRequest(shards)
	if err != nil {
		return err
	}

	if rbc.contentLength == 0 {
		rbc.contentLength = uint64(len(data))
	}

	// send to me
	if err := rbc.handleValueRequest(rbc.proposer, reqs[0].(*ValRequest)); err != nil {
		return err
	}

	if err := rbc.distributeMessage(rbc.proposer, reqs[1:]); err != nil {
		return err
	}

	return nil
}

// HandleMessage will used in ACS
func (rbc *RBC) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	req, err := processMessage(msg)
	if err != nil {
		return err
	}

	r := request{
		sender: sender,
		data:   req,
		err:    make(chan error),
	}

	if rbc.contentLength == 0 {
		rbc.contentLength = msg.Rbc.ContentLength
	}

	if rbc.contentLength != msg.Rbc.ContentLength {
		return errors.New(fmt.Sprintf("inavlid content length - know as : %d, receive : %d", rbc.contentLength, msg.Rbc.ContentLength))
	}

	rbc.reqChan <- r
	return <-r.err
}

// handleMessage will distinguish input message (from ACS)
func (rbc *RBC) muxRequest(sender cleisthenes.Member, req cleisthenes.Request) error {
	switch r := req.(type) {
	case *ValRequest:
		return rbc.handleValueRequest(sender, r)
	case *EchoRequest:
		return rbc.handleEchoRequest(sender, r)
	case *ReadyRequest:
		return rbc.handleReadyRequest(sender, r)
	default:
		return ErrInvalidRBCType
	}
}

func (rbc *RBC) handleValueRequest(sender cleisthenes.Member, req *ValRequest) error {
	if rbc.valReceived.Value() {
		return errors.New("already receive req message")
	}

	if rbc.echoSent.Value() {
		return errors.New(fmt.Sprintf("already sent echo message - sender id : %s", sender.Address.String()))
	}

	if !validateMessage(req) {
		return errors.New("invalid VALUE request")
	}

	rbc.valReceived.Set(true)
	rbc.echoSent.Set(true)
	echoReq := &EchoRequest{*req}
	rbc.shareMessage(rbc.proposer, echoReq)

	iLogger.Infof(nil, "[VAL] owner : %s, proposer : %s, sender : %s", rbc.owner.Address.String(), rbc.proposer.Address.String(), sender.Address.String())
	return nil
}

func (rbc *RBC) handleEchoRequest(sender cleisthenes.Member, req *EchoRequest) error {
	if req, _ := rbc.echoReqRepo.Find(sender.Address); req != nil {
		return errors.New(fmt.Sprintf("already received echo request - from : %s", sender.Address.String()))
	}

	if !validateMessage(&req.ValRequest) {
		return errors.New(fmt.Sprintf("invalid ECHO request - from : %s", sender.Address.String()))
	}

	if err := rbc.echoReqRepo.Save(sender.Address, req); err != nil {
		return err
	}

	if rbc.countEchos(req.RootHash) >= rbc.echoThreshold() && !rbc.readySent.Value() {
		rbc.readySent.Set(true)
		readyReq := &ReadyRequest{req.RootHash}
		rbc.shareMessage(rbc.proposer, readyReq)
	}

	if rbc.countReadys(req.RootHash) >= rbc.outputThreshold() && rbc.countEchos(req.RootHash) >= rbc.echoThreshold() {
		value, err := rbc.interpolate(req.RootHash)
		if err != nil {
			return err
		}
		rbc.lock.Lock()
		rbc.value = value
		rbc.lock.Unlock()
	}

	iLogger.Infof(nil, "[ECHO] owner : %s, proposer : %s, sender : %s", rbc.owner.Address.String(), rbc.proposer.Address.String(), sender.Address.String())
	return nil
}

func (rbc *RBC) handleReadyRequest(sender cleisthenes.Member, req *ReadyRequest) error {
	if req, _ := rbc.readyReqRepo.Find(sender.Address); req != nil {
		return errors.New(fmt.Sprintf("already received ready request - from : %s", sender.Address.String()))
	}

	if err := rbc.readyReqRepo.Save(sender.Address, req); err != nil {
		return err
	}

	if rbc.countReadys(req.RootHash) >= rbc.readyThreshold() && !rbc.readySent.Value() {
		rbc.readySent.Set(true)
		readyReq := &ReadyRequest{req.RootHash}
		rbc.shareMessage(rbc.proposer, readyReq)
	}

	if rbc.countReadys(req.RootHash) >= rbc.outputThreshold() && rbc.countEchos(req.RootHash) >= rbc.echoThreshold() {
		value, err := rbc.interpolate(req.RootHash)
		if err != nil {
			return err
		}
		rbc.lock.Lock()
		rbc.value = value
		rbc.lock.Unlock()
	}

	iLogger.Infof(nil, "[READY] owner : %s, proposer : %s, sender : %s", rbc.owner.Address.String(), rbc.proposer.Address.String(), sender.Address.String())
	return nil
}

// Return output
func (rbc *RBC) Value() []byte {
	rbc.lock.Lock()
	defer rbc.lock.Unlock()

	if rbc.value != nil {
		value := rbc.value
		rbc.value = nil
		return value
	}
	return nil
}

func (rbc *RBC) run() {
	for {
		select {
		case stop := <-rbc.closeChan:
			rbc.closeChan <- stop
			return
		case req := <-rbc.reqChan:
			req.err <- rbc.muxRequest(req.sender, req.data)
		}
	}
}

func (rbc *RBC) Close() {
	rbc.closeChan <- struct{}{}
	<-rbc.closeChan
}

func (rbc *RBC) countEchos(rootHash []byte) int {
	cnt := 0

	reqs := rbc.echoReqRepo.FindAll()
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*EchoRequest).RootHash) {
			cnt++
		}
	}

	return cnt
}

func (rbc *RBC) countReadys(rootHash []byte) int {
	cnt := 0

	reqs := rbc.readyReqRepo.FindAll()
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*ReadyRequest).RootHash) {
			cnt++
		}
	}

	return cnt
}

// interpolate the given shards
// if try to interpolate not enough ( < N - 2f ) shards then return error
func (rbc *RBC) interpolate(rootHash []byte) ([]byte, error) {
	reqs := rbc.echoReqRepo.FindAll()

	if len(reqs) < rbc.numDataShards {
		return nil, errors.New(fmt.Sprintf("not enough shards - minimum : %d, got : %d ", rbc.numDataShards, len(reqs)))
	}

	// To indicate missing data, you should set the shard to nil before calling Reconstruct
	shards := make([][]byte, rbc.numDataShards+rbc.numParityShards)
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*EchoRequest).RootHash) {
			order := merkletree.OrderOfData(req.(*EchoRequest).Indexes)
			shards[order] = req.(*EchoRequest).Data.Bytes()
		}
	}

	if err := rbc.enc.Reconstruct(shards); err != nil {
		return nil, err
	}

	// TODO : check interpolated data's merkle root hash and request's merkle root hash

	var value []byte
	for _, data := range shards[:rbc.numDataShards] {
		value = append(value, data...)
	}

	return value[:rbc.contentLength], nil
}

// wait until receive N - f ECHO messages
func (rbc *RBC) echoThreshold() int {
	return rbc.n - rbc.f
}

func (rbc *RBC) readyThreshold() int {
	return rbc.f + 1
}

func (rbc *RBC) outputThreshold() int {
	return 2*rbc.f + 1
}

func makeRequest(shards []merkletree.Data) ([]cleisthenes.Request, error) {
	tree, err := merkletree.New(shards)
	if err != nil {
		return nil, err
	}

	reqs := make([]cleisthenes.Request, 0)
	rootHash := tree.MerkleRoot()
	for _, shard := range shards {
		paths, indexes, err := tree.MerklePath(shard)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, &ValRequest{
			RootHash: rootHash,
			Data:     shard,
			RootPath: paths,
			Indexes:  indexes,
		})
	}

	return reqs, nil
}

func processMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	switch msg.Rbc.Type {
	case pb.RBC_VAL:
		return processValueMessage(msg)
	case pb.RBC_ECHO:
		return processEchoMessage(msg)
	case pb.RBC_READY:
		return processReadyMessage(msg)
	default:
		return nil, errors.New("error processing message with invalid type")
	}
}

func processValueMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	req := &ValRequest{}
	if err := json.Unmarshal(msg.Rbc.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

func processEchoMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	req := &EchoRequest{}
	if err := json.Unmarshal(msg.Rbc.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

func processReadyMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	req := &ReadyRequest{}
	if err := json.Unmarshal(msg.Rbc.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

// validate given value message and echo message
func validateMessage(req *ValRequest) bool {
	return merkletree.ValidatePath(req.Data, req.RootHash, req.RootPath, req.Indexes)
}

// make shards using reed-solomon erasure coding
func shard(enc reedsolomon.Encoder, data []byte) ([]merkletree.Data, error) {
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}
	if err := enc.Encode(shards); err != nil {
		return nil, err
	}

	dataList := make([]merkletree.Data, 0)

	for _, shard := range shards {
		dataList = append(dataList, merkletree.NewData(shard))
	}

	return dataList, nil
}
