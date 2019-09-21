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
	mu    sync.RWMutex
	items map[string]interface{}
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
			items: make(map[string]interface{}),
		}
	}

	return sm
}

func (sm *StrMap) _getShard(key string) *strMapShard {
	return sm.shards[memHashString(key)%sm.shardCount]
}

// Store ...
func (sm *StrMap) Store(key string, item interface{}) {
	var (
		shard = sm._getShard(key)
		ok    bool
	)
	shard.mu.RLock()
	if _, ok = shard.items[key]; ok {
		shard.mu.RUnlock()
		// Already inserted here. Since that means it already
		// passed through this operation, no need to do the
		// extra book keeping, and we can return right now
		return
	}
	shard.mu.RUnlock()
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, ok = shard.items[key]; !ok {
		shard.items[key] = item
	}
}

// Delete ...
func (sm *StrMap) Delete(key string) {
	shard := sm._getShard(key)
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
}

// Get ...
func (sm *StrMap) Get(key string) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	item, ok := shard.items[key]
	return item, ok
}
