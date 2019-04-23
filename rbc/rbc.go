package rbc

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/klauspost/reedsolomon"
)

type RBC struct {

	// number of network nodes
	n int

	// number of byzantine nodes which can tolerate
	f int

	proposer cleisthenes.Member

	// Erasure coding using reed-solomon method
	enc reedsolomon.Encoder

	// Broadcast message after when rbc receive message
	messages []*pb.Message

	// Request of other rbcs
	valueReqRepo cleisthenes.RequestRepository
	echoReqRepo  cleisthenes.RequestRepository
	readyReqRepo cleisthenes.RequestRepository

	// internal channels to communicate with other components
	closeChan   chan struct{}
	messageChan chan InnerMessage
	requestChan chan InnerRequest

	broadCaster cleisthenes.Broadcaster
}

func NewRBC(config cleisthenes.Config) *RBC {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) broadcast(msg *pb.Message) error {
	panic("implement me w/ test case :-)")
}

// HandleMessage will used in ACS
func (rbc *RBC) HandleMessage(senderId cleisthenes.ConnId, msg *pb.Message) error {
	panic("implement me w/ test case :-)")
}

// handleMessage will distinguish input message (from ACS)
func (rbc *RBC) handleMessage(senderId cleisthenes.ConnId, msg *pb.Message) error {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) handleValueRequest(senderId cleisthenes.ConnId, req ValRequest) error {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) handleEchoRequest(senderId cleisthenes.ConnId, req EchoRequest) error {
	panic("implement me w/ test case :-)")
}

func (rbc *RBC) handleReadyRequest(senderId cleisthenes.ConnId, req ReadyRequest) error {
	panic("implement me w/ test case :-)")
}

// Return output
func (rbc *RBC) Value() []byte {
	panic("implement me w/ test case :-)")
}

// Return messages
func (rbc *RBC) Messages() []*pb.Message {
	panic("implement me w/ test case :-)")
}

func (r *RBC) run() {
	panic("implement me w/ test case :-)")
}

func (r *RBC) stop() {
	panic("implement me w/ test case :-)")
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
func shard(enc reedsolomon.Encoder, data []byte) ([][]byte, error) {
	panic("implement me w/ test case :-)")
}
