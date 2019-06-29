package honeybadger

import (
	"errors"
	"fmt"
	"sync"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

type ACS interface {
	HandleInput(data []byte) error
	HandleMessage(sender cleisthenes.Member, msg *pb.Message) error
}

type acsRepository struct {
	lock  sync.RWMutex
	items map[cleisthenes.Epoch]ACS
}

func newACSRepository() *acsRepository {
	return &acsRepository{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Epoch]ACS),
	}
}

func (r *acsRepository) save(epoch cleisthenes.Epoch, instance ACS) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.items[epoch]
	if ok {
		return errors.New(fmt.Sprintf("acs instance with epoch [%d]", epoch))
	}
	r.items[epoch] = instance
	return nil
}

func (r *acsRepository) find(epoch cleisthenes.Epoch) (ACS, bool) {
	result, ok := r.items[epoch]
	return result, ok
}

type acsFactory struct{}

func (f *acsFactory) create() (ACS, error) {
	// TODO: create ACS instance
	return nil, nil
}
