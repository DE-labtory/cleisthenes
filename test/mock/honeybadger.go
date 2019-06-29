package mock

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
)

type HoneyBadger struct {
	HandleContributionFunc func(contribution cleisthenes.Contribution) error
	HandleMessageFunc      func(msg *pb.Message) error
}

func (hb *HoneyBadger) HandleContribution(contribution cleisthenes.Contribution) error {
	return nil
}
func (hb *HoneyBadger) HandleMessage(msg *pb.Message) error {
	return nil
}
