package rbc

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/rbc/merkletree"
	"github.com/klauspost/reedsolomon"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var ErrInvalidRBCType = errors.New("Invalid RBC message type")

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

	valReceived, echoSent, readySent bool

	// internal channels to communicate with other components
	closeChan chan struct{}
	reqChan   chan request
	inputChan chan InputMessage

	broadcaster cleisthenes.Broadcaster
}

func New(n, f int, owner, proposer cleisthenes.Member, broadcaster cleisthenes.Broadcaster) *RBC {
	return nil
}

func (rbc *RBC) broadcast(proposer cleisthenes.Member, typ pb.RBCType, req cleisthenes.Request) error {
	return nil
}

// MakeRequest make requests to send to other nodes
// it is used in ACS
func (rbc *RBC) MakeRequest(data []byte) ([]cleisthenes.Request, error) {
	shards, err := shard(rbc.enc, data)
	if err != nil {
		return nil, err
	}

	reqs, err := makeRequest(shards)
	if err != nil {
		return nil, err
	}

	if rbc.contentLength == 0 {
		rbc.contentLength = uint64(len(data))
	}

	if err := rbc.handleValueRequest(rbc.proposer, reqs[0].(*ValRequest)); err != nil {
		return nil, err
	}

	return reqs[1:], nil
}

// HandleMessage will used in ACS
func (rbc *RBC) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	req := request{
		sender: sender,
		data:   msg,
		err:    make(chan error),
	}

	if rbc.contentLength == 0 {
		rbc.contentLength = msg.Rbc.ContentLength
	}

	if rbc.contentLength != msg.Rbc.ContentLength {
		return fmt.Errorf("inavlid content length - know as : %d, receive : %d", rbc.contentLength, msg.Rbc.ContentLength)
	}

	rbc.reqChan <- req
	return <-req.err
}

// handleMessage will distinguish input message (from ACS)
func (rbc *RBC) muxRequest(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	switch msg.Rbc.Type {
	case pb.RBC_VAL:
		var req ValRequest
		err := json.Unmarshal(msg.Rbc.Payload, &req)
		if err != nil {
			return err
		}
		return rbc.handleValueRequest(sender, &req)
	case pb.RBC_ECHO:
		var req EchoRequest
		err := json.Unmarshal(msg.Rbc.Payload, &req)
		if err != nil {
			return err
		}
		return rbc.handleEchoRequest(sender, &req)
	case pb.RBC_READY:
		var req ReadyRequest
		err := json.Unmarshal(msg.Rbc.Payload, &req)
		if err != nil {
			return err
		}
		return rbc.handleReadyRequest(sender, &req)
	default:
		return ErrInvalidRBCType
	}
}

func (rbc *RBC) handleValueRequest(sender cleisthenes.Member, req *ValRequest) error {
	return nil
}

func (rbc *RBC) handleEchoRequest(sender cleisthenes.Member, req *EchoRequest) error {
	return nil
}

func (rbc *RBC) handleReadyRequest(sender cleisthenes.Member, req *ReadyRequest) error {
	return nil
}

// Return output
func (rbc *RBC) Value() []byte {
	return rbc.value
}

func (r *RBC) run() {
	for {
		select {
		case stop := <-r.closeChan:
			r.closeChan <- stop
			return
		case req := <-r.reqChan:
			req.err <- r.muxRequest(req.sender, req.data)
		}
	}
}

func (r *RBC) close() {
	r.closeChan <- struct{}{}
	<-r.closeChan
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

func (rbc *RBC) countEchos(rootHash []byte) int {
	return 0
}

func (rbc *RBC) countReadys(rootHash []byte) int {
	return 0
}

// interpolate the given shards
// if try to interpolate not enough ( < N - 2f ) shards then return error
func (rbc *RBC) interpolate(rootHash []byte) ([]byte, error) {
	reqs := rbc.echoReqRepo.FindAll()

	if len(reqs) < rbc.numDataShards {
		return nil, fmt.Errorf("not enough shards - minimum : %d, got : %d ", rbc.numDataShards, len(reqs))
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
