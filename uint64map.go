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
	shard := sm._getShard(key)
	shard.mu.Lock()
	shard.values[key] = value
	shard.mu.Unlock()
}

// Load ...
func (sm *Uint64Map) Load(key uint64) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	value, ok := shard.values[key]
	shard.mu.RUnlock()
	return value, ok
}

// LoadOrStore ...
func (sm *Uint64Map) LoadOrStore(key uint64, value interface{}) (actual interface{}, loaded bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	// Fast path assuming value has a somewhat high chance of already being
	// there.
	if actual, loaded = shard.values[key]; loaded {
		shard.mu.RUnlock()
		return
	}
	shard.mu.RUnlock()
	shard.mu.Lock()
	// Gotta check again, unfortunately
	if actual, loaded = shard.values[key]; loaded {
		shard.mu.Unlock()
		return
	}
	shard.values[key] = value
	shard.mu.Unlock()
	return value, loaded
}

// Delete ...
func (sm *Uint64Map) Delete(key uint64) {
	shard := sm._getShard(key)
	shard.mu.Lock()
	delete(shard.values, key)
	shard.mu.Unlock()
}

// Range is modeled after sync.Map.Range. It calls f sequentially for each key
// and value present in each of the shards in the map. If f returns false, range
// stops the iteration.
//
// No key will be visited more than once, but if any value is inserted
// concurrently, Range may or may not visit it. Similarly, if a value is
// modified concurrently, Range may visit the previous or newest version of said
// value. Notice that this is RLocking, don't modify values directly here.
func (sm *Uint64Map) Range(f func(key uint64, value interface{}) bool) {
	for _, shard := range sm.shards {
		shard.mu.RLock()
		for key, value := range shard.values {
			if !f(key, value) {
				shard.mu.RUnlock()
				return
			}
		}
		shard.mu.RUnlock()
	}
}
