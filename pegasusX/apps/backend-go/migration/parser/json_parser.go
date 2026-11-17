package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// StreamJSON reads either a JSON array of records ([{...}]) OR a composite JSON object whose
// array properties represent distinct named streams (e.g. {"products": [...], "stock": [...]}),
// deterministically flattening nested maps with dot-separated keys.
func StreamJSON(defaultStreamName string, data []byte, onChunk StreamRowHandler) ([]ParsedStream, error) {
	clean := bytes.TrimSpace(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))
	if len(clean) == 0 {
		return nil, fmt.Errorf("%w: empty json artifact", ErrMalformedArtifact)
	}

	var root any
	dec := json.NewDecoder(bytes.NewReader(clean))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("%w: invalid json: %v", ErrMalformedArtifact, err)
	}

	switch typed := root.(type) {
	case []any:
		stream, err := emitJSONArrayStream(fallbackStreamName(defaultStreamName, "records"), typed, onChunk)
		if err != nil {
			return nil, err
		}
		return []ParsedStream{stream}, nil
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for k, v := range typed {
			if _, isArr := v.([]any); isArr {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			// Single JSON object record
			stream, err := emitJSONArrayStream(fallbackStreamName(defaultStreamName, "record"), []any{typed}, onChunk)
			if err != nil {
				return nil, err
			}
			return []ParsedStream{stream}, nil
		}
		sort.Strings(keys)
		streams := make([]ParsedStream, 0, len(keys))
		for _, k := range keys {
			arr := typed[k].([]any)
			if len(arr) == 0 {
				continue
			}
			stream, err := emitJSONArrayStream(k, arr, onChunk)
			if err != nil {
				return nil, err
			}
			streams = append(streams, stream)
		}
		if len(streams) == 0 {
			return nil, fmt.Errorf("%w: json arrays contain zero rows", ErrMalformedArtifact)
		}
		return streams, nil
	default:
		return nil, fmt.Errorf("%w: json root must be object or array", ErrMalformedArtifact)
	}
}

func emitJSONArrayStream(streamName string, items []any, onChunk StreamRowHandler) (ParsedStream, error) {
	flatRows := make([]map[string]any, 0, len(items))
	headerSet := make(map[string]struct{})
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			m = map[string]any{"value": item}
		}
		flat := make(map[string]any)
		flattenJSONMap("", m, flat)
		if len(flat) == 0 {
			continue
		}
		for k := range flat {
			headerSet[k] = struct{}{}
		}
		flatRows = append(flatRows, flat)
	}

	headers := make([]string, 0, len(headerSet))
	for h := range headerSet {
		headers = append(headers, h)
	}
	sort.Strings(headers)

	meta := ParsedStream{
		StreamName: streamName,
		Format:     FormatJSON,
		Encoding:   "utf-8",
		Headers:    headers,
		TotalRows:  int64(len(flatRows)),
	}

	chunk := make([]ParsedRow, 0, defaultStreamChunkSize)
	for idx, flat := range flatRows {
		pr := ParsedRow{RowIndex: int64(idx), Fields: flat}
		if len(meta.SampleRows) < 20 {
			meta.SampleRows = append(meta.SampleRows, pr)
		}
		if onChunk != nil {
			chunk = append(chunk, pr)
			if len(chunk) >= defaultStreamChunkSize {
				if err := onChunk(streamName, headers, chunk); err != nil {
					return ParsedStream{}, err
				}
				chunk = chunk[:0]
			}
		}
	}
	if onChunk != nil && len(chunk) > 0 {
		if err := onChunk(streamName, headers, chunk); err != nil {
			return ParsedStream{}, err
		}
	}
	return meta, nil
}

func flattenJSONMap(prefix string, src map[string]any, dst map[string]any) {
	for k, v := range src {
		key := strings.TrimSpace(k)
		if prefix != "" {
			key = prefix + "." + key
		}
		switch val := v.(type) {
		case map[string]any:
			flattenJSONMap(key, val, dst)
		case []any:
			// Summarize nested primitive arrays compactly; flatten bounded nested objects
			allObj := true
			for _, elem := range val {
				if _, ok := elem.(map[string]any); !ok {
					allObj = false
					break
				}
			}
			if allObj && len(val) > 0 {
				for i, elem := range val {
					if i >= 5 {
						// Preserve full payload as raw JSON beyond 5 items so columns stay bounded
						b, _ := json.Marshal(val)
						dst[key] = string(b)
						break
					}
					flattenJSONMap(fmt.Sprintf("%s[%d]", key, i), elem.(map[string]any), dst)
				}
			} else {
				b, _ := json.Marshal(val)
				dst[key] = string(b)
			}
		case json.Number:
			dst[key] = val.String()
		default:
			dst[key] = val
		}
	}
}
