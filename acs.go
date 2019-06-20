package cleisthenes

import "github.com/DE-labtory/cleisthenes/pb"

type ACS interface {
	HandleInput(data []byte) error
	HandleMessage(sender Member, msg *pb.Message) error
	Close()
}
