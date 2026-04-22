package workers

import (
	"bytes"
	"sync"
)

// BufPool is a sync.Pool for *bytes.Buffer instances, eliminating per-request
// heap allocations on hot paths (JSON marshalling, Kafka emission, HTTP
// response capture). Get() returns a reset buffer; Put() returns it to the pool.
var BufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 512))
	},
}

// GetBuffer retrieves a buffer from the pool, pre-reset.
func GetBuffer() *bytes.Buffer {
	buf := BufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the pool. Oversized buffers (>64 KB) are
// discarded to prevent the pool from holding onto large allocations.
func PutBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 65536 {
		return // let GC reclaim oversized buffers
	}
	BufPool.Put(buf)
}
