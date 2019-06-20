package honeybadger

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

type ACSMap struct {
	lock  sync.WaitGroup
	items map[cleisthenes.Epoch]cleisthenes.ACS
}
