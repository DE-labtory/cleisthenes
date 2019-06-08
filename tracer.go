package cleisthenes

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"

	"github.com/DE-labtory/iLogger"
)

type Tracer interface {
	Log(keyvals ...string)
	Trace()
}

type MemCacheTracer struct {
	lock      sync.RWMutex
	logger    log.Logger
	traceList []string
}

func NewMemCacheTracer() *MemCacheTracer {
	return &MemCacheTracer{
		lock:      sync.RWMutex{},
		traceList: make([]string, 0),
	}
}

func (t *MemCacheTracer) Log(keyvals ...string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(keyvals) == 0 {
		return
	}
	if len(keyvals)%2 == 1 {
		keyvals = append(keyvals, "")
	}

	kvs := make([]string, 0)

	for i := 0; i < len(keyvals); i += 2 {
		k, v := keyvals[i], keyvals[i+1]
		kvs = append(kvs, fmt.Sprintf("%s=%s", k, v))
	}
	trace := strings.Join(kvs, " ")
	t.traceList = append(t.traceList, trace)
}

func (t *MemCacheTracer) Trace() {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, trace := range t.traceList {
		iLogger.Info(nil, trace)
	}
}
