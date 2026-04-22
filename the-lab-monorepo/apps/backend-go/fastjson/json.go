// Package fastjson provides a drop-in jsoniter alias for hot paths.
// Import "backend-go/fastjson" and call fastjson.Marshal / fastjson.Unmarshal
// where throughput matters (WebSocket broadcasts, Kafka emitters, cache middleware).
// Cold paths can keep using encoding/json — no migration needed.
package fastjson

import jsoniter "github.com/json-iterator/go"

// JSON is the jsoniter instance configured for maximum compatibility with stdlib.
var JSON = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	Marshal       = JSON.Marshal
	Unmarshal     = JSON.Unmarshal
	MarshalIndent = JSON.MarshalIndent
	NewDecoder    = JSON.NewDecoder
	NewEncoder    = JSON.NewEncoder
)
