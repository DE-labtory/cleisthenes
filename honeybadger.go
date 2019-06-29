package cleisthenes

import "github.com/DE-labtory/cleisthenes/pb"

type Epoch uint64

type HoneyBadger interface {
	HandleContribution(contribution Contribution) error
	HandleMessage(msg *pb.Message) error
}
