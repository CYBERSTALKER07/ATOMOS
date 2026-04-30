// Package rpc holds cross-cutting gRPC infrastructure used by all internal
// service clients and servers in backend-go. It does not contain business
// logic; each domain owns its own sub-package (e.g. rpc/optimizer).
//
// Codec choice: we register a JSON codec so all internal RPCs share the same
// wire format as the existing HTTP calls — zero re-serialization cost when
// migrating HTTP → gRPC and no protoc toolchain dependency at build time.
// The codec is registered once at package init via encoding.RegisterCodec.
package rpc

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

const CodecName = "json"

func init() {
	encoding.RegisterCodec(JSONCodec{})
}

// JSONCodec implements grpc/encoding.Codec with standard library JSON.
// gRPC allows any codec; the client and server must agree on the name.
type JSONCodec struct{}

func (JSONCodec) Name() string { return CodecName }

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
