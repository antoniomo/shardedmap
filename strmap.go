package shardedmap

import (
	"sync"
)

// Implementation: This is a sharded map so that the cost of locking is
// distributed with the data, instead of a single lock.
// The optimal number of shards will probably depend on the number of system
// cores but we provide a general default.
type StrMap struct {
	shardCount uint64 // Don't alter after creation, no mutex here
	shards     []*strMapShard
}

type strMapShard struct {
	mu     sync.RWMutex
	values map[string]interface{}
}

// NewStrMap ...
func NewStrMap(shardCount int) *StrMap {

	if shardCount <= 0 {
		shardCount = defaultShards
	}

	sm := &StrMap{
		shardCount: uint64(shardCount),
		shards:     make([]*strMapShard, shardCount),
	}

	for i := range sm.shards {
		sm.shards[i] = &strMapShard{
			values: make(map[string]interface{}),
		}
	}

	return sm
}

func (sm *StrMap) _getShard(key string) *strMapShard {
	return sm.shards[memHashString(key)%sm.shardCount]
}

// Store ...
func (sm *StrMap) Store(key string, value interface{}) {
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
func (sm *StrMap) Load(key string) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	value, ok := shard.values[key]
	return value, ok
}

// LoadOrStore ...
func (sm *StrMap) LoadOrStore(key string, value interface{}) (actual interface{}, loaded bool) {
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
func (sm *StrMap) Delete(key string) {
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
// value.
func (sm *StrMap) Range(f func(key string, value interface{}) bool) {
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
