package hotspot

import (
	"hash/fnv"

	"github.com/google/uuid"
)

const DefaultShardCount = 16

var shardCount = DefaultShardCount

func ConfigureShardCount(count int) {
	if count > 0 {
		shardCount = count
	}
}

func ShardCount() int {
	return shardCount
}

func AllShards() []int64 {
	shards := make([]int64, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = int64(i)
	}
	return shards
}

func ShardForKey(key string) int64 {
	if shardCount <= 1 {
		return 0
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	return int64(hasher.Sum32() % uint32(shardCount))
}

func NewOpaqueID() string {
	return uuid.NewString()
}

func NewOrderID() string {
	return NewOpaqueID()
}

func NewInvoiceID() string {
	return NewOpaqueID()
}

func NewPredictionID() string {
	return NewOpaqueID()
}

func NewPredictionItemID() string {
	return NewOpaqueID()
}
