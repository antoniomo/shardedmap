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
	mu    sync.RWMutex
	items map[uint64]interface{}
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
			items: make(map[uint64]interface{}),
		}
	}

	return sm
}

func (sm *Uint64Map) _getShard(key uint64) *uint64MapShard {
	return sm.shards[key%sm.shardCount]
}

// Insert ...
func (sm *Uint64Map) Insert(key uint64, item interface{}) {
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
func (sm *Uint64Map) Delete(key uint64) {
	shard := sm._getShard(key)
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
}

// Get ...
func (sm *Uint64Map) Get(key uint64) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	item, ok := shard.items[key]
	return item, ok
}
