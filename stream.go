package cleisthenes

import (
	"context"

	"github.com/DE-labtory/cleisthenes/pb"
	"google.golang.org/grpc"
)

type Stream interface {
	Send(message *pb.Message) error
	Recv() (*pb.Message, error)
}

type StreamWrapper interface {
	Stream
	Close()
	GetStream() Stream
}

// client side stream wrapper
type CStreamWrapper struct {
	conn         *grpc.ClientConn
	client       pb.StreamServiceClient
	clientStream pb.StreamService_MessageStreamClient
	cancel       context.CancelFunc
}

func NewClientStreamWrapper(conn *grpc.ClientConn) (StreamWrapper, error) {
	panic("implement me w/ test case :-)")
}

func (csw *CStreamWrapper) Send(message *pb.Message) error {
	panic("implement me w/ test case :-)")
}

func (csw *CStreamWrapper) Recv() (*pb.Message, error) {
	panic("implement me w/ test case :-)")
}

func (csw *CStreamWrapper) Close() {
	panic("implement me w/ test case :-)")
}

func (csw *CStreamWrapper) GetStream() Stream {
	panic("implement me w/ test case :-)")
}

// server side stream wrapper
type SStreamWrapper struct {
	ServerStream pb.StreamService_MessageStreamServer
	cancel       context.CancelFunc
}

func NewServerStreamWrapper(serverStream pb.StreamService_MessageStreamServer, cancel context.CancelFunc) StreamWrapper {
	panic("implement me w/ test case :-)")
}

func (ssw *SStreamWrapper) Send(message *pb.Message) error {
	panic("implement me w/ test case :-)")
}

func (ssw *SStreamWrapper) Recv() (*pb.Message, error) {
	panic("implement me w/ test case :-)")
}

func (ssw *SStreamWrapper) Close() {
	panic("implement me w/ test case :-)")
}

func (ssw *SStreamWrapper) GetStream() Stream {
	panic("implement me w/ test case :-)")
}
