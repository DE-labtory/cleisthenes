package cleisthenes

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/DE-labtory/iLogger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnHandler func(conn Connection)
type ErrHandler func(err error)

type GrpcServer struct {
	connHandler ConnHandler
	errHandler  ErrHandler
	addr        Address
	lis         net.Listener
}

func NewServer(addr Address) *GrpcServer {
	return &GrpcServer{
		addr: addr,
	}
}

// MessageStream handle request to connection from remote peer. First,
// configure ip of remote peer, then based on that info create connection
// after that handler of server process handles that connection.
func (s GrpcServer) MessageStream(streamServer pb.StreamService_MessageStreamServer) error {
	ip := extractRemoteAddr(streamServer)

	_, cancel := context.WithCancel(context.Background())
	streamWrapper := NewServerStreamWrapper(streamServer, cancel)

	conn, err := NewConnection(Address{Ip: ip}, uuid.New().String(), streamWrapper)
	if err != nil {
		return err
	}
	if s.connHandler != nil {
		s.connHandler(conn)
	}
	return nil
}

// extractRemoteAddr returns address of attached peer
func extractRemoteAddr(stream pb.StreamService_MessageStreamServer) string {
	p, ok := peer.FromContext(stream.Context())
	if !ok {
		return ""
	}
	if p.Addr == nil {
		return ""
	}
	return p.Addr.String()
}

func (s *GrpcServer) OnConn(handler ConnHandler) {
	if handler == nil {
		return
	}
	s.connHandler = handler
}

func (s *GrpcServer) OnErr(handler ErrHandler) {
	if handler == nil {
		return
	}
	s.errHandler = handler
}

func (s *GrpcServer) Listen() {
	lis, err := net.Listen("tcp", s.addr.String())
	if err != nil {
		iLogger.Errorf(nil, "listen error: %s", err.Error())
	}
	defer lis.Close()

	g := grpc.NewServer()
	defer g.Stop()

	pb.RegisterStreamServiceServer(g, s)
	reflection.Register(g)
	s.lis = lis
	iLogger.Infof(nil, "listen ... on: [%s]", s.addr.String())

	if err := g.Serve(lis); err != nil {
		iLogger.Errorf(nil, "listen error: [%s]", err.Error())
		g.Stop()
		lis.Close()
	}
}

func (s *GrpcServer) Stop() {
	if s.lis != nil {
		s.lis.Close()
	}
}

const (
	DefaultDialTimeout = 3 * time.Second
)

type DialOpts struct {
	// Ip is target address which grpc client is going to dial
	Addr Address

	// Duration for which to block while established a new connection
	Timeout time.Duration
}

type GrpcClient struct{}

func NewClient() *GrpcClient {
	return &GrpcClient{}
}

func (c GrpcClient) Dial(opts DialOpts) (Connection, error) {
	dialContext, _ := context.WithTimeout(context.Background(), opts.Timeout)
	gconn, err := grpc.DialContext(dialContext, opts.Addr.String(), append([]grpc.DialOption{}, grpc.WithInsecure())...)
	if err != nil {
		return nil, err
	}
	streamWrapper, err := NewClientStreamWrapper(gconn)
	if err != nil {
		return nil, err
	}
	conn, err := NewConnection(opts.Addr, uuid.New().String(), streamWrapper)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
