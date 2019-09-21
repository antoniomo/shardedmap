package shardedmap

import (
	"sync"
)

// Implementation: This is a sharded map so that the cost of locking is
// distributed with the data, instead of a single lock.
// The optimal number of shards will probably depend on the number of system
// cores but we provide a general default.
type Uint64Map struct {
	shardCount uint64 // Don't alter after creation, no mutex here
	shards     []*uint64MapShard
}

type uint64MapShard struct {
	mu     sync.RWMutex
	values map[uint64]interface{}
}

// NewUint64Map ...
func NewUint64Map(shardCount int) *Uint64Map {

	if shardCount <= 0 {
		shardCount = defaultShards
	}

	sm := &Uint64Map{
		shardCount: uint64(shardCount),
		shards:     make([]*uint64MapShard, shardCount),
	}

	for i := range sm.shards {
		sm.shards[i] = &uint64MapShard{
			values: make(map[uint64]interface{}),
		}
	}

	return sm
}

func (sm *Uint64Map) _getShard(key uint64) *uint64MapShard {
	return sm.shards[key%sm.shardCount]
}

// Store ...
func (sm *Uint64Map) Store(key uint64, value interface{}) {
	var (
		shard = sm._getShard(key)
		ok    bool
	)
	shard.mu.RLock()
	if _, ok = shard.values[key]; ok {
		shard.mu.RUnlock()
		// Already inserted here. Since that means it already
		// passed through this operation, no need to do the
		// extra book keeping, and we can return right now
		return
	}
	shard.mu.RUnlock()
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok = shard.values[key]; !ok {
		shard.values[key] = value
	}
}

// Load ...
func (sm *Uint64Map) Load(key uint64) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	value, ok := shard.values[key]
	return value, ok
}

// LoadOrStore ...
func (sm *Uint64Map) LoadOrStore(key uint64, value interface{}) (actual interface{}, loaded bool) {
	var (
		shard = sm._getShard(key)
	)
	shard.mu.RLock()
	if actual, loaded = shard.values[key]; loaded {
		shard.mu.RUnlock()
		return
	}
	shard.mu.RUnlock()
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if actual, loaded = shard.values[key]; !loaded {
		shard.values[key] = value
		return value, loaded
	}
	return actual, loaded
}

// Delete ...
func (sm *Uint64Map) Delete(key uint64) {
	shard := sm._getShard(key)
	shard.mu.Lock()
	delete(shard.values, key)
	shard.mu.Unlock()
}
