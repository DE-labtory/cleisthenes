package rbc

import (
	"encoding/json"

	"github.com/DE-labtory/cleisthenes/rbc/merkletree"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/klauspost/reedsolomon"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var ErrInvalidRBCType = errors.New("Invalid RBC message type")

type RBC struct {
	// number of network nodes
	n int

	// number of byzantine nodes which can tolerate
	f int

	proposer cleisthenes.ConnId

	// Erasure coding using reed-solomon method
	enc reedsolomon.Encoder

	// output of RBC
	value []byte

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

func NewRBC(config cleisthenes.Config) *RBC {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) broadcast(msg request) error {
	panic("implement me w/ test case :-)")
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

	if err := rbc.handleValueRequest(rbc.proposer, reqs[0].(*ValRequest)); err != nil {
		return nil, err
	}

	return reqs[1:], nil
}

// HandleMessage will used in ACS
func (rbc *RBC) HandleMessage(connId cleisthenes.ConnId, msg *pb.Message_Rbc) error {
	req := request{
		senderId: connId,
		data:     msg,
		err:      make(chan error),
	}

	rbc.reqChan <- req
	return <-req.err
}

// handleMessage will distinguish input message (from ACS)
func (rbc *RBC) muxRequest(connId cleisthenes.ConnId, msg *pb.Message_Rbc) error {
	switch msg.Rbc.Type {
	case pb.RBC_VAL:
		var req ValRequest
		err := json.Unmarshal(msg.Rbc.Payload, &req)
		if err != nil {
			return err
		}
		return rbc.handleValueRequest(connId, &req)
	case pb.RBC_ECHO:
		var req EchoRequest
		err := json.Unmarshal(msg.Rbc.Payload, &req)
		if err != nil {
			return err
		}
		return rbc.handleEchoRequest(connId, &req)
	case pb.RBC_READY:
		var req ReadyRequest
		err := json.Unmarshal(msg.Rbc.Payload, &req)
		if err != nil {
			return err
		}
		return rbc.handleReadyRequest(connId, &req)
	default:
		return ErrInvalidRBCType
	}
}

func (rbc *RBC) handleValueRequest(connId cleisthenes.ConnId, req *ValRequest) error {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) handleEchoRequest(connId cleisthenes.ConnId, req *EchoRequest) error {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) handleReadyRequest(connId cleisthenes.ConnId, req *ReadyRequest) error {
	panic("implement me w/ test case :-)")
}

// Return output
func (rbc *RBC) Value() []byte {
	panic("implement me w/ test case :-)")
}

func (r *RBC) run() {
	for {
		select {
		case stop := <-r.closeChan:
			r.closeChan <- stop
			return
		case req := <-r.reqChan:
			req.err <- r.muxRequest(req.senderId, req.data)
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

// interpolate the given shards
// if try to interpolate not enough ( < N - 2f ) shards then return error
func interpolate(rootHash []byte, shards [][]byte) ([]byte, error) {
	panic("implement me w/ test case :-)")
}

// validate given echo message
func validateMessage(echo *EchoRequest) bool {
	panic("implement me w/ test case :-)")
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
