package rbc

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type (
	ValRequest struct {
		RootHash []byte
		Branch   []byte
		Block    [][]byte
	}

	EchoRequest struct {
		ValRequest
	}

	ReadyRequest struct {
		RootHash []byte
	}
)

// It means it is abstracted as Request interface (cleisthenes/request.go)
func (r ValRequest) Recv()   {}
func (r EchoRequest) Recv()  {}
func (r ReadyRequest) Recv() {}

// Received request
type (
	ValReqRepository struct {
		recv map[cleisthenes.ConnId]*ValRequest
	}

	EchoReqRepository struct {
		recv map[cleisthenes.ConnId]*EchoRequest
	}

	ReadyReqRepository struct {
		recv map[cleisthenes.ConnId]*ReadyRequest
	}
)

type (
	// InnerMessage used in channel when received message
	InnerMessage struct {
		senderId cleisthenes.ConnId
		msg      *pb.Message
		err      error
	}

	// InnerRequest used in channel when send messages
	InnerRequest struct {
		msgs []*pb.Message
		err  error
	}
)
