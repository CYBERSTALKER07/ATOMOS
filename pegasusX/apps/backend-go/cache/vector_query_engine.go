package cache

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// RQEDialectVersion is the mandatory Redis Query Engine dialect for vector and modern searches (redis-query-engine).
const RQEDialectVersion = 2

// VectorAlgorithm specifies HNSW (approximate, >10k vectors) or FLAT (exact, <10k vectors).
type VectorAlgorithm string

const (
	VectorAlgorithmHNSW VectorAlgorithm = "HNSW"
	VectorAlgorithmFLAT VectorAlgorithm = "FLAT"
)

// VectorDistanceMetric specifies the distance metric matching the embedding model.
type VectorDistanceMetric string

const (
	DistanceMetricCosine VectorDistanceMetric = "COSINE"
	DistanceMetricIP     VectorDistanceMetric = "IP"
	DistanceMetricL2     VectorDistanceMetric = "L2"
)

// RQEFieldType defines the narrowest index field type for the access pattern (redis-query-engine).
type RQEFieldType string

const (
	RQEFieldText    RQEFieldType = "TEXT"
	RQEFieldTag     RQEFieldType = "TAG"
	RQEFieldNumeric RQEFieldType = "NUMERIC"
	RQEFieldGeo     RQEFieldType = "GEO"
	RQEFieldVector  RQEFieldType = "VECTOR"
)

// RQEFieldSchema describes one indexed field in an FT.CREATE schema.
type RQEFieldSchema struct {
	Name           string
	Type           RQEFieldType
	Sortable       bool
	Unnormalized   bool // UNF for TAG SORTABLE
	Weight         float64
	VectorAlgo     VectorAlgorithm
	VectorDim      int
	DistanceMetric VectorDistanceMetric
	HNSWM          int // Connections per node (default 16)
	HNSWEfConst    int // Build-time graph quality (default 200)
	HNSWEfRuntime  int // Query-time candidate list size (default 50)
}

// RQEIndexSpec defines an FT.CREATE index specification on HASH or JSON documents.
type RQEIndexSpec struct {
	IndexName       string
	AliasName       string
	OnJSON          bool
	Prefixes        []string
	SkipInitialScan bool
	Fields          []RQEFieldSchema
}

// BuildFTCreateArgs builds the exact FT.CREATE command arguments following redis-query-engine
// and redis-vector-search best practices (mandatory PREFIX, narrow field types, HNSW/FLAT tuning).
func BuildFTCreateArgs(spec RQEIndexSpec) ([]any, error) {
	idx := strings.TrimSpace(spec.IndexName)
	if idx == "" {
		return nil, fmt.Errorf("rqe: index name required")
	}
	var prefixes []string
	for _, p := range spec.Prefixes {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			prefixes = append(prefixes, trimmed)
		}
	}
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("rqe: at least one key prefix is required to prevent full-database indexing")
	}
	if len(spec.Fields) == 0 {
		return nil, fmt.Errorf("rqe: at least one schema field is required")
	}

	onType := "HASH"
	if spec.OnJSON {
		onType = "JSON"
	}

	args := []any{"FT.CREATE", idx, "ON", onType, "PREFIX", len(prefixes)}
	for _, p := range prefixes {
		args = append(args, p)
	}
	if spec.SkipInitialScan {
		args = append(args, "SKIPINITIALSCAN")
	}
	args = append(args, "SCHEMA")

	for _, f := range spec.Fields {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			return nil, fmt.Errorf("rqe: field name required")
		}
		args = append(args, name, string(f.Type))
		switch f.Type {
		case RQEFieldText:
			if f.Weight > 0 {
				args = append(args, "WEIGHT", f.Weight)
			}
			if f.Sortable {
				args = append(args, "SORTABLE")
			}
		case RQEFieldTag:
			if f.Sortable {
				args = append(args, "SORTABLE")
				if f.Unnormalized {
					args = append(args, "UNF")
				}
			}
		case RQEFieldNumeric:
			if f.Sortable {
				args = append(args, "SORTABLE")
			}
		case RQEFieldGeo:
			// No extra modifiers needed for standard GEO points
		case RQEFieldVector:
			if f.VectorDim <= 0 {
				return nil, fmt.Errorf("rqe: vector field %q requires positive DIM", name)
			}
			algo := f.VectorAlgo
			if algo == "" {
				algo = VectorAlgorithmHNSW
			}
			metric := f.DistanceMetric
			if metric == "" {
				metric = DistanceMetricCosine
			}
			vecAttrs := []any{
				"TYPE", "FLOAT32",
				"DIM", f.VectorDim,
				"DISTANCE_METRIC", string(metric),
			}
			if algo == VectorAlgorithmHNSW {
				if f.HNSWM > 0 {
					vecAttrs = append(vecAttrs, "M", f.HNSWM)
				}
				if f.HNSWEfConst > 0 {
					vecAttrs = append(vecAttrs, "EF_CONSTRUCTION", f.HNSWEfConst)
				}
				if f.HNSWEfRuntime > 0 {
					vecAttrs = append(vecAttrs, "EF_RUNTIME", f.HNSWEfRuntime)
				}
			}
			args = append(args, string(algo), len(vecAttrs))
			args = append(args, vecAttrs...)
		default:
			return nil, fmt.Errorf("rqe: unsupported field type %q", f.Type)
		}
	}
	return args, nil
}

// NumericRange represents an inclusive numeric filter [@field:[min max]].
type NumericRange struct {
	Min float64
	Max float64
}

// HybridVectorQuery specifies a pre-filtered KNN vector search under DIALECT 2 (redis-vector-search).
type HybridVectorQuery struct {
	IndexOrAlias   string
	VectorField    string
	Vector         []float32
	TopK           int
	TagFilters     map[string]string
	NumericFilters map[string]NumericRange
	ReturnFields   []string
}

// BuildHybridKNNQuery constructs a DIALECT 2 FT.SEARCH command that applies TAG and NUMERIC
// pre-filters before the KNN vector similarity comparison.
func BuildHybridKNNQuery(q HybridVectorQuery) ([]any, error) {
	idx := strings.TrimSpace(q.IndexOrAlias)
	if idx == "" {
		return nil, fmt.Errorf("rqe: index or alias required")
	}
	vecField := strings.TrimSpace(q.VectorField)
	if vecField == "" {
		vecField = "embedding"
	}
	if len(q.Vector) == 0 {
		return nil, fmt.Errorf("rqe: non-empty query vector required")
	}
	topK := q.TopK
	if topK <= 0 {
		topK = 5
	}

	filterExpr := buildPreFilterExpression(q.TagFilters, q.NumericFilters)
	knnQuery := fmt.Sprintf("%s=>[KNN %d @%s $vec AS vector_distance]", filterExpr, topK, vecField)

	args := []any{
		"FT.SEARCH", idx, knnQuery,
		"PARAMS", 2, "vec", EncodeFloat32VectorBlob(q.Vector),
		"SORTBY", "vector_distance", "ASC",
		"LIMIT", 0, topK,
	}
	if len(q.ReturnFields) > 0 {
		fields := append([]string(nil), q.ReturnFields...)
		hasDist := false
		for _, f := range fields {
			if f == "vector_distance" {
				hasDist = true
				break
			}
		}
		if !hasDist {
			fields = append(fields, "vector_distance")
		}
		args = append(args, "RETURN", len(fields))
		for _, f := range fields {
			args = append(args, f)
		}
	}
	args = append(args, "DIALECT", RQEDialectVersion)
	return args, nil
}

func buildPreFilterExpression(tags map[string]string, nums map[string]NumericRange) string {
	var parts []string
	if len(tags) > 0 {
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			val := escapeTagValue(strings.TrimSpace(tags[k]))
			if k != "" && val != "" {
				parts = append(parts, fmt.Sprintf("@%s:{%s}", k, val))
			}
		}
	}
	if len(nums) > 0 {
		keys := make([]string, 0, len(nums))
		for k := range nums {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rng := nums[k]
			parts = append(parts, fmt.Sprintf("@%s:[%s %s]",
				k,
				strconv.FormatFloat(rng.Min, 'f', -1, 64),
				strconv.FormatFloat(rng.Max, 'f', -1, 64),
			))
		}
	}
	if len(parts) == 0 {
		return "*"
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func escapeTagValue(v string) string {
	var b strings.Builder
	for _, r := range v {
		switch r {
		case ',', '.', '<', '>', '{', '}', '[', ']', '"', '\'', ':', ';', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '-', '+', '=', '~', ' ':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// EncodeFloat32VectorBlob encodes a float32 slice into an IEEE-754 little-endian byte slice
// expected by RediSearch FLOAT32 VECTOR fields.
func EncodeFloat32VectorBlob(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// DecodeFloat32VectorBlob decodes an IEEE-754 little-endian byte slice back into a float32 slice.
func DecodeFloat32VectorBlob(blob []byte) []float32 {
	if len(blob)%4 != 0 || len(blob) == 0 {
		return nil
	}
	out := make([]float32, len(blob)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return out
}

// EnsureIndexWithAlias creates the RQE index (ignoring "Index already exists") and points
// the alias to it via FT.ALIASUPDATE for zero-downtime schema rollouts (redis-query-engine).
func EnsureIndexWithAlias(ctx context.Context, client *redis.Client, spec RQEIndexSpec) error {
	if client == nil {
		return fmt.Errorf("rqe: nil redis client")
	}
	args, err := BuildFTCreateArgs(spec)
	if err != nil {
		return err
	}
	if err := client.Do(ctx, args...).Err(); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "index already exists") {
			return fmt.Errorf("rqe ft.create %s: %w", spec.IndexName, err)
		}
	}
	if alias := strings.TrimSpace(spec.AliasName); alias != "" {
		if err := client.Do(ctx, "FT.ALIASUPDATE", alias, spec.IndexName).Err(); err != nil {
			return fmt.Errorf("rqe ft.aliasupdate %s -> %s: %w", alias, spec.IndexName, err)
		}
	}
	return nil
}
