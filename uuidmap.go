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
	mu    sync.RWMutex
	items map[UUID]interface{}
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
			items: make(map[UUID]interface{}),
		}
	}

	return sm
}

func (sm *UUIDMap) _getShard(key UUID) *uuidMapShard {
	return sm.shards[memHash(key[:])%sm.shardCount]
}

// Insert ...
func (sm *UUIDMap) Insert(key UUID, item interface{}) {
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
func (sm *UUIDMap) Delete(key UUID) {
	shard := sm._getShard(key)
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
}

// Get ...
func (sm *UUIDMap) Get(key UUID) (interface{}, bool) {
	shard := sm._getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	item, ok := shard.items[key]
	return item, ok
}
