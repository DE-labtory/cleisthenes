package cleisthenes

import "github.com/DE-labtory/cleisthenes/pb"

type Epoch uint64

type HoneyBadger interface {
	HandleBatch(batch Batch) error
	HandleMessage(msg *pb.Message) error
}
