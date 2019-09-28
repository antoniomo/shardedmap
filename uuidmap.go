package shardedmap

import (
	"sync"
)

// Implementation: This is a sharded map so that the cost of locking is
// distributed with the data, instead of a single lock.
// The optimal number of shards will probably depend on the number of system
// cores but we provide a general default.
type UUIDMap struct {
	shardCount uint64 // Don't alter after creation, no mutex here
	shards     []*uuidMapShard
}

type uuidMapShard struct {
	mu     sync.RWMutex
	values map[UUID]interface{}
}

type UUID [16]byte

// NewUUIDMap ...
func NewUUIDMap(shardCount int) *UUIDMap {
	if shardCount <= 0 {
		shardCount = defaultShards
	}

	sm := &UUIDMap{
		shardCount: uint64(shardCount),
		shards:     make([]*uuidMapShard, shardCount),
	}

	for i := range sm.shards {
		sm.shards[i] = &uuidMapShard{
			values: make(map[UUID]interface{}),
		}
	}

	return sm
}

func (sm *UUIDMap) _getShard(key UUID) *uuidMapShard {
	return sm.shards[memHash(key[:])&(sm.shardCount-1)]
}

// Store ...
func (sm *UUIDMap) Store(key UUID, value interface{}) {
	shard := sm._getShard(key)
	shard.mu.Lock()
	shard.values[key] = value
	shard.mu.Unlock()
}

// Load ...
func (sm *UUIDMap) Load(key UUID) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	value, ok := shard.values[key]
	shard.mu.RUnlock()
	return value, ok
}

// LoadOrStore ...
func (sm *UUIDMap) LoadOrStore(key UUID, value interface{}) (actual interface{}, loaded bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	// Fast path assuming value has a somewhat high chance of already being
	// there.
	if actual, loaded = shard.values[key]; loaded {
		shard.mu.RUnlock()
		return
	}
	shard.mu.RUnlock()
	// Gotta check again, unfortunately
	shard.mu.Lock()
	if actual, loaded = shard.values[key]; loaded {
		shard.mu.Unlock()
		return
	}
	shard.values[key] = value
	shard.mu.Unlock()
	return actual, loaded
}

// Delete ...
func (sm *UUIDMap) Delete(key UUID) {
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
func (sm *UUIDMap) Range(f func(key UUID, value interface{}) bool) {
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
