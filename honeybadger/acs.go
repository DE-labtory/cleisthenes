package honeybadger

import (
	"errors"
	"fmt"
	"sync"

	"github.com/DE-labtory/cleisthenes/config"

	"github.com/DE-labtory/cleisthenes/acs"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

type ACS interface {
	HandleInput(data []byte) error
	HandleMessage(sender cleisthenes.Member, msg *pb.Message) error
	Close()
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
		return errors.New(fmt.Sprintf("acs instance already exist with epoch [%d]", epoch))
	}
	r.items[epoch] = instance
	return nil
}

func (r *acsRepository) find(epoch cleisthenes.Epoch) (ACS, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	result, ok := r.items[epoch]
	return result, ok
}

func (r *acsRepository) delete(epoch cleisthenes.Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.items, epoch)
}

// ACSFactory helps create ACS instance easily. To create ACS, we need lots of DI
// And for the ease of creating ACS, ACSFactory have components which is need to
// create ACS
type ACSFactory interface {
	Create(epoch cleisthenes.Epoch) (ACS, error)
}

type DefaultACSFactory struct {
	n              int
	f              int
	acsOwner       cleisthenes.Member
	batchSender    cleisthenes.BatchSender
	memberMap      cleisthenes.MemberMap
	dataReceiver   cleisthenes.DataReceiver
	dataSender     cleisthenes.DataSender
	binaryReceiver cleisthenes.BinaryReceiver
	binarySender   cleisthenes.BinarySender
	broadcaster    cleisthenes.Broadcaster
}

func NewDefaultACSFactory(
	n int,
	f int,
	acsOwner cleisthenes.Member,
	memberMap cleisthenes.MemberMap,
	dataReceiver cleisthenes.DataReceiver,
	dataSender cleisthenes.DataSender,
	binaryReceiver cleisthenes.BinaryReceiver,
	binarySender cleisthenes.BinarySender,
	batchSender cleisthenes.BatchSender,
	broadcaster cleisthenes.Broadcaster,
) *DefaultACSFactory {
	return &DefaultACSFactory{
		n:              n,
		f:              f,
		acsOwner:       acsOwner,
		memberMap:      memberMap,
		dataReceiver:   dataReceiver,
		dataSender:     dataSender,
		binaryReceiver: binaryReceiver,
		binarySender:   binarySender,
		batchSender:    batchSender,
		broadcaster:    broadcaster,
	}
}

func (f *DefaultACSFactory) Create(epoch cleisthenes.Epoch) (ACS, error) {
	dataChan := cleisthenes.NewDataChannel(f.n)
	binaryChan := cleisthenes.NewBinaryChannel(f.n)

	return acs.New(
		config.Get().HoneyBadger.NetworkSize,
		config.Get().HoneyBadger.Byzantine,
		epoch,
		f.acsOwner,
		f.memberMap,
		dataChan,
		dataChan,
		binaryChan,
		binaryChan,
		f.batchSender,
		f.broadcaster,
	)
}
