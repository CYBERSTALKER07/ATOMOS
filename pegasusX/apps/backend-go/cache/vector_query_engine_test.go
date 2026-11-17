package cache

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildFTCreateArgs_AndVectorBlobRoundTrip(t *testing.T) {
	spec := RQEIndexSpec{
		IndexName:       "idx:semantic_v1",
		AliasName:       "idx:semantic",
		Prefixes:        []string{"semantic:entry:"},
		SkipInitialScan: false,
		Fields: []RQEFieldSchema{
			{Name: "prompt", Type: RQEFieldText, Weight: 2.0},
			{Name: "task_type", Type: RQEFieldTag, Sortable: true, Unnormalized: true},
			{Name: "tenant_id", Type: RQEFieldTag, Sortable: true, Unnormalized: true},
			{Name: "created_at", Type: RQEFieldNumeric, Sortable: true},
			{
				Name:           "embedding",
				Type:           RQEFieldVector,
				VectorAlgo:     VectorAlgorithmHNSW,
				VectorDim:      4,
				DistanceMetric: DistanceMetricCosine,
				HNSWM:          16,
				HNSWEfConst:    200,
				HNSWEfRuntime:  50,
			},
		},
	}

	args, err := BuildFTCreateArgs(spec)
	if err != nil {
		t.Fatalf("BuildFTCreateArgs error: %v", err)
	}
	joined := fmt.Sprint(args)
	for _, want := range []string{
		"FT.CREATE idx:semantic_v1 ON HASH PREFIX 1 semantic:entry:",
		"task_type TAG SORTABLE UNF",
		"embedding VECTOR HNSW 12 TYPE FLOAT32 DIM 4 DISTANCE_METRIC COSINE M 16 EF_CONSTRUCTION 200 EF_RUNTIME 50",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in %q", want, joined)
		}
	}

	vec := []float32{0.25, -0.5, 0.75, 1.0}
	blob := EncodeFloat32VectorBlob(vec)
	decoded := DecodeFloat32VectorBlob(blob)
	if len(decoded) != len(vec) {
		t.Fatalf("decoded len = %d, want %d", len(decoded), len(vec))
	}
	for i := range vec {
		if decoded[i] != vec[i] {
			t.Fatalf("decoded[%d] = %v, want %v", i, decoded[i], vec[i])
		}
	}
}

func TestBuildHybridKNNQuery_Dialect2AndPreFilter(t *testing.T) {
	args, err := BuildHybridKNNQuery(HybridVectorQuery{
		IndexOrAlias: "idx:semantic",
		VectorField:  "embedding",
		Vector:       []float32{0.1, 0.2, 0.3, 0.4},
		TopK:         5,
		TagFilters: map[string]string{
			"task_type": "dispatch_optimization",
			"tenant_id": "sup-123",
		},
		NumericFilters: map[string]NumericRange{
			"created_at": {Min: 1700000000, Max: 1800000000},
		},
		ReturnFields: []string{"prompt", "response"},
	})
	if err != nil {
		t.Fatalf("BuildHybridKNNQuery error: %v", err)
	}
	joined := fmt.Sprint(args)
	if !strings.Contains(joined, "(@task_type:{dispatch_optimization} @tenant_id:{sup\\-123} @created_at:[1700000000 1800000000])=>[KNN 5 @embedding $vec AS vector_distance]") {
		t.Fatalf("unexpected query filter in %q", joined)
	}
	if !strings.HasSuffix(joined, "DIALECT 2]") {
		t.Fatalf("expected DIALECT 2 suffix in %q", joined)
	}
}
