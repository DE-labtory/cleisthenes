package cleisthenes

import (
	"net"
	"time"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnHandler func(conn Connection)
type ErrHandler func(err error)

type GrpcServer struct {
	connHandler ConnHandler
	errHandler  ErrHandler
	ip          string
	lis         net.Listener
}

func NewServer() *GrpcServer {
	return &GrpcServer{}
}

func (s GrpcServer) MessageStream(streamServer pb.StreamService_MessageStreamServer) error {
	panic("implement me w/ test case :-)")
}

func (s *GrpcServer) OnConn(handler ConnHandler) {
	panic("implement me w/ test case :-)")
}

func (s *GrpcServer) OnErr(handler ErrHandler) {
	panic("implement me w/ test case :-)")
}

func (s *GrpcServer) Listen(ip string) {
	panic("implement me w/ test case :-)")
}

func (s *GrpcServer) Stop() {
	panic("implement me w/ test case :-)")
}

type DialOpts struct {
	// Ip is target address which grpc client is going to dial
	Ip string

	// Duration for which to block while established a new connection
	Timeout time.Duration
}

type GrpcClient struct {
	// Options for setting up new connections
	DialOpts DialOpts
}

func (c GrpcClient) Dial(serverIp string) (Connection, error) {
	panic("implement me w/ test case :-)")
}
