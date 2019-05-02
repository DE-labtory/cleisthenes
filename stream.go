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
	ctx, cf := context.WithCancel(context.Background())
	streamserviceClient := pb.NewStreamServiceClient(conn)
	clientStream, err := streamserviceClient.MessageStream(ctx)

	if err != nil {
		return nil, err
	}

	return &CStreamWrapper{
		conn:         conn,
		client:       streamserviceClient,
		clientStream: clientStream,
		cancel:       cf,
	}, nil
}

func (csw *CStreamWrapper) Send(message *pb.Message) error {
	return csw.clientStream.Send(message)
}

func (csw *CStreamWrapper) Recv() (*pb.Message, error) {
	return csw.clientStream.Recv()
}

func (csw *CStreamWrapper) Close() {
	csw.conn.Close()
	csw.clientStream.CloseSend()
	csw.cancel()
}

func (csw *CStreamWrapper) GetStream() Stream {
	return csw.clientStream
}

// server side stream wrapper
type SStreamWrapper struct {
	ServerStream pb.StreamService_MessageStreamServer
	cancel       context.CancelFunc
}

func NewServerStreamWrapper(serverStream pb.StreamService_MessageStreamServer, cancel context.CancelFunc) StreamWrapper {
	return &SStreamWrapper{
		cancel:       cancel,
		ServerStream: serverStream,
	}
}

func (ssw *SStreamWrapper) Send(message *pb.Message) error {
	return ssw.ServerStream.Send(message)
}

func (ssw *SStreamWrapper) Recv() (*pb.Message, error) {
	return ssw.ServerStream.Recv()
}

func (ssw *SStreamWrapper) Close() {
	ssw.cancel()
}

func (ssw *SStreamWrapper) GetStream() Stream {
	return ssw.ServerStream
}
