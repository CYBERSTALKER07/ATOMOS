package retailer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	h3 "github.com/uber/h3-go/v4"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

const (
	// DeliveryPerimeterKey is the canonical Redis set checked by order creation
	// via SISMEMBER for O(1) retailer zone gating.
	DeliveryPerimeterKey = "ssmr:delivery_perimeter"
	// DeliveryPerimeterCompactedKey stores the compacted perimeter for
	// transport/debug parity with PolygonToCells + CompactCells output.
	DeliveryPerimeterCompactedKey = "ssmr:delivery_perimeter:compacted"

	DefaultPerimeterResolution = 9
	defaultPolygonVertices     = 64
)

var (
	// ErrZoneMiss is returned when the retailer coordinate falls outside the
	// cached delivery perimeter OR when perimeter cache state is unavailable.
	ErrZoneMiss = errors.New("zone_miss")
	// ErrPerimeterUnavailable marks missing/unreachable perimeter cache state.
	ErrPerimeterUnavailable = errors.New("delivery_perimeter_unavailable")
)

// PerimeterSetStore is the narrow Redis set contract required by spatial checks.
// RedisBackend satisfies this interface.
type PerimeterSetStore interface {
	ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error
	SIsMember(ctx context.Context, key string, member string) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// RetailerProximityConfig controls perimeter keys and H3 resolution.
type RetailerProximityConfig struct {
	PerimeterKey          string
	CompactedPerimeterKey string
	Resolution            int
	Log                   *slog.Logger
}

// PerimeterSnapshot summarizes precomputed perimeter write results.
type PerimeterSnapshot struct {
	Cells          int
	CompactedCells int
	Resolution     int
}

// RetailerProximityService owns server-side H3 derivation and Redis-backed
// zone membership checks.
type RetailerProximityService struct {
	store                 PerimeterSetStore
	perimeterKey          string
	compactedPerimeterKey string
	resolution            int
	log                   *slog.Logger
}

// NewRetailerProximityService constructs the proximity service.
func NewRetailerProximityService(store PerimeterSetStore, cfg RetailerProximityConfig) *RetailerProximityService {
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if strings.TrimSpace(cfg.PerimeterKey) == "" {
		cfg.PerimeterKey = DeliveryPerimeterKey
	}
	if strings.TrimSpace(cfg.CompactedPerimeterKey) == "" {
		cfg.CompactedPerimeterKey = DeliveryPerimeterCompactedKey
	}
	if cfg.Resolution <= 0 {
		cfg.Resolution = DefaultPerimeterResolution
	}
	return &RetailerProximityService{
		store:                 store,
		perimeterKey:          cfg.PerimeterKey,
		compactedPerimeterKey: cfg.CompactedPerimeterKey,
		resolution:            cfg.Resolution,
		log:                   cfg.Log,
	}
}

// CellForCoordinate derives the canonical H3 index for a retailer coordinate.
func (s *RetailerProximityService) CellForCoordinate(lat, lng float64) (string, error) {
	if lat == 0 && lng == 0 {
		return "", fmt.Errorf("lat/lng required")
	}
	cell, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, s.resolution)
	if err != nil {
		return "", fmt.Errorf("lat_lng_to_cell: %w", err)
	}
	if !cell.IsValid() {
		return "", fmt.Errorf("lat_lng_to_cell produced invalid cell")
	}
	return cell.String(), nil
}

// PerimeterReady reports whether the canonical perimeter key exists.
func (s *RetailerProximityService) PerimeterReady(ctx context.Context) (bool, error) {
	if s == nil || s.store == nil {
		return false, ErrPerimeterUnavailable
	}
	ready, err := s.store.Exists(ctx, s.perimeterKey)
	if err != nil {
		return false, fmt.Errorf("check perimeter key: %w", err)
	}
	return ready, nil
}

// IsRetailerInZone checks O(1) zone membership using Redis SISMEMBER.
// When the direct cell misses, expands k-rings up to proximity.DefaultNeighborK().
func (s *RetailerProximityService) IsRetailerInZone(ctx context.Context, h3Index string) error {
	if strings.TrimSpace(h3Index) == "" {
		return fmt.Errorf("%w: empty h3 index", ErrZoneMiss)
	}
	if s == nil || s.store == nil {
		return fmt.Errorf("%w: %w", ErrZoneMiss, ErrPerimeterUnavailable)
	}

	ready, err := s.PerimeterReady(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZoneMiss, err)
	}
	if !ready {
		return fmt.Errorf("%w: %w", ErrZoneMiss, ErrPerimeterUnavailable)
	}

	if member, err := s.store.SIsMember(ctx, s.perimeterKey, h3Index); err != nil {
		return fmt.Errorf("%w: perimeter sismember failed: %v", ErrZoneMiss, err)
	} else if member {
		return nil
	}

	// Neighbor ring fallback: parse cell and expand GridDisk rings.
	var center h3.Cell
	if err := center.UnmarshalText([]byte(h3Index)); err != nil {
		return fmt.Errorf("%w: h3_index=%s", ErrZoneMiss, h3Index)
	}
	for k := 1; k <= proximity.DefaultNeighborK(); k++ {
		ring, err := h3.GridDisk(center, k)
		if err != nil {
			break
		}
		for _, cell := range ring {
			if !cell.IsValid() {
				continue
			}
			if cell.String() == h3Index {
				continue
			}
			member, err := s.store.SIsMember(ctx, s.perimeterKey, cell.String())
			if err != nil {
				return fmt.Errorf("%w: perimeter sismember failed: %v", ErrZoneMiss, err)
			}
			if member {
				return nil
			}
		}
	}

	return fmt.Errorf("%w: h3_index=%s", ErrZoneMiss, h3Index)
}

// PrecomputeDeliveryZoneForCenter builds a circular polygon and stores both the
// expanded and compacted perimeter sets with TTL=0 persistence semantics.
func (s *RetailerProximityService) PrecomputeDeliveryZoneForCenter(ctx context.Context, lat, lng, radiusKm float64) (PerimeterSnapshot, error) {
	polygon := circularPolygon(lat, lng, radiusKm, defaultPolygonVertices)
	return s.PrecomputeDeliveryZone(ctx, polygon)
}

// PrecomputeDeliveryZone computes polygon coverage at service resolution,
// compacts the cells, then writes expanded + compacted sets to Redis.
func (s *RetailerProximityService) PrecomputeDeliveryZone(ctx context.Context, polygon [][2]float64) (PerimeterSnapshot, error) {
	if s == nil || s.store == nil {
		return PerimeterSnapshot{}, ErrPerimeterUnavailable
	}
	cells, err := polygonToCells(polygon, s.resolution)
	if err != nil {
		return PerimeterSnapshot{}, err
	}
	if len(cells) == 0 {
		return PerimeterSnapshot{}, fmt.Errorf("polygon_to_cells returned empty set")
	}

	compacted, err := compactCells(cells)
	if err != nil {
		return PerimeterSnapshot{}, err
	}

	if err := s.store.ReplaceSet(ctx, s.perimeterKey, cells, 0); err != nil {
		return PerimeterSnapshot{}, fmt.Errorf("replace perimeter set: %w", err)
	}
	if err := s.store.ReplaceSet(ctx, s.compactedPerimeterKey, compacted, 0); err != nil {
		return PerimeterSnapshot{}, fmt.Errorf("replace compacted perimeter set: %w", err)
	}

	s.log.Info("delivery perimeter precomputed",
		"cells", len(cells),
		"compacted_cells", len(compacted),
		"resolution", s.resolution,
	)

	return PerimeterSnapshot{
		Cells:          len(cells),
		CompactedCells: len(compacted),
		Resolution:     s.resolution,
	}, nil
}

func polygonToCells(polygon [][2]float64, resolution int) ([]string, error) {
	if len(polygon) < 3 {
		return nil, fmt.Errorf("polygon must contain at least 3 points")
	}
	loop := make(h3.GeoLoop, 0, len(polygon))
	for _, point := range polygon {
		loop = append(loop, h3.NewLatLng(point[0], point[1]))
	}
	indexes, err := h3.PolygonToCells(h3.GeoPolygon{GeoLoop: loop}, resolution)
	if err != nil {
		return nil, fmt.Errorf("polygon_to_cells: %w", err)
	}

	seen := make(map[string]struct{}, len(indexes))
	cells := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		if !idx.IsValid() {
			continue
		}
		cell := idx.String()
		if _, ok := seen[cell]; ok {
			continue
		}
		seen[cell] = struct{}{}
		cells = append(cells, cell)
	}
	sort.Strings(cells)
	return cells, nil
}

func compactCells(cells []string) ([]string, error) {
	indexes := make([]h3.Cell, 0, len(cells))
	for _, cellID := range cells {
		var idx h3.Cell
		if err := idx.UnmarshalText([]byte(cellID)); err != nil {
			return nil, fmt.Errorf("invalid h3 cell: %s", cellID)
		}
		if !idx.IsValid() {
			return nil, fmt.Errorf("invalid h3 cell: %s", cellID)
		}
		indexes = append(indexes, idx)
	}
	compacted, err := h3.CompactCells(indexes)
	if err != nil {
		return nil, fmt.Errorf("compact_cells: %w", err)
	}
	out := make([]string, 0, len(compacted))
	for _, idx := range compacted {
		if !idx.IsValid() {
			continue
		}
		out = append(out, idx.String())
	}
	sort.Strings(out)
	return out, nil
}

func circularPolygon(lat, lng, radiusKm float64, vertices int) [][2]float64 {
	if vertices < 12 {
		vertices = 12
	}
	if radiusKm <= 0 {
		radiusKm = 1
	}

	latDelta := radiusKm / 111.0
	cosLat := math.Cos(lat * math.Pi / 180.0)
	if math.Abs(cosLat) < 0.01 {
		cosLat = 0.01
	}
	lngDelta := radiusKm / (111.0 * cosLat)

	polygon := make([][2]float64, 0, vertices)
	for i := 0; i < vertices; i++ {
		angle := 2 * math.Pi * float64(i) / float64(vertices)
		pLat := lat + latDelta*math.Sin(angle)
		pLng := lng + lngDelta*math.Cos(angle)
		polygon = append(polygon, [2]float64{pLat, pLng})
	}
	return polygon
}
