package acs

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

type binaryStateMap struct {
	lock  sync.RWMutex
	items map[cleisthenes.Address]cleisthenes.BinaryState
}

type broadcastDataMap struct {
	lock  sync.RWMutex
	items map[cleisthenes.Address][]byte
}
