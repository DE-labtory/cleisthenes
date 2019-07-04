package acs

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

type binaryStateMap struct {
	lock  sync.RWMutex
	items map[cleisthenes.Member]cleisthenes.BinaryState
}

func NewBinaryStateMap() binaryStateMap {
	return binaryStateMap{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Member]cleisthenes.BinaryState),
	}
}

func (b *binaryStateMap) set(member cleisthenes.Member, bin cleisthenes.BinaryState) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.items[member] = bin
}

func (b *binaryStateMap) item(member cleisthenes.Member) cleisthenes.BinaryState {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.items[member]
}

func (b *binaryStateMap) itemMap() map[cleisthenes.Member]cleisthenes.BinaryState {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.items
}

func (b *binaryStateMap) exist(member cleisthenes.Member) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	_, ok := b.items[member]
	return ok
}

type broadcastDataMap struct {
	lock  sync.RWMutex
	items map[cleisthenes.Member][]byte
}

func NewbroadcastDataMap() broadcastDataMap {
	return broadcastDataMap{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Member][]byte),
	}
}

func (b *broadcastDataMap) set(member cleisthenes.Member, data []byte) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.items[member] = data
}

func (b *broadcastDataMap) item(member cleisthenes.Member) []byte {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.items[member]
}

func (b *broadcastDataMap) exist(member cleisthenes.Member) bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	_, ok := b.items[member]
	return ok
}
