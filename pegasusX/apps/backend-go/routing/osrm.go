package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

const (
	osrmRouteTimeout = 3 * time.Second
	osrmTableTimeout = 10 * time.Second

	// Keep OSRM POST table requests and responses safely under the 4 MiB wire cap
	// while staying within standard OSRM default --max-table-size=100 coordinates.
	osrmMaxTablePayloadBytes = (4 << 20) - (64 << 10)
	osrmDefaultChunkCoords   = 50
	osrmMaxSingleTableCoords = 100
	osrmTableWorkerLimit     = 8
)

// OSRMClient calls the OSRM route and table services for driving geometry and matrices.
type OSRMClient struct {
	baseURL           string
	http              *http.Client
	breaker           *circuit.Breaker
	tableChunkCoords  int
	maxTableBodyBytes int
}

// NewOSRMClient returns nil when baseURL is empty.
func NewOSRMClient(baseURL string, breaker *circuit.Breaker) *OSRMClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &OSRMClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: osrmRouteTimeout},
		breaker: breaker,
	}
}

type osrmRouteResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Geometry string    `json:"geometry"`
		Distance float64   `json:"distance"`
		Legs     []osrmLeg `json:"legs"`
	} `json:"routes"`
}

type osrmTableResponse struct {
	Code     string      `json:"code"`
	Distances [][]float64 `json:"distances"`
}

// RouteGeometry requests a driving route through OSRM for the given waypoints.
func (c *OSRMClient) RouteGeometry(ctx context.Context, routeID string, waypoints []LatLng, includeSteps bool) (RouteGeometry, error) {
	if c == nil {
		return RouteGeometry{}, fmt.Errorf("osrm: nil client")
	}
	if len(waypoints) < 2 {
		return RouteGeometry{
			RouteID:     routeID,
			Coordinates: []LatLng{},
			Source:      "insufficient_waypoints",
			StopCount:   len(waypoints),
		}, nil
	}

	var geometry RouteGeometry
	call := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			c.routeURL(waypoints, includeSteps),
			nil,
		)
		if err != nil {
			return fmt.Errorf("osrm request: %w", err)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("osrm http: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return fmt.Errorf("osrm read body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("osrm status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var payload osrmRouteResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return fmt.Errorf("osrm decode: %w", err)
		}
		if !strings.EqualFold(payload.Code, "Ok") || len(payload.Routes) == 0 {
			return fmt.Errorf("osrm code %q", payload.Code)
		}
		route := payload.Routes[0]
		coords, err := DecodePolyline(route.Geometry)
		if err != nil {
			return fmt.Errorf("osrm polyline: %w", err)
		}
		if len(coords) < 2 {
			return fmt.Errorf("osrm polyline too short")
		}

		geometry = RouteGeometry{
			RouteID:         routeID,
			EncodedPolyline: route.Geometry,
			Coordinates:     coords,
			Source:          "osrm_driving",
			StopCount:       len(waypoints),
		}
		if includeSteps {
			geometry.Steps = parseOSRMSteps(route.Legs)
		}
		return nil
	}

	var err error
	if c.breaker != nil {
		err = c.breaker.Do(ctx, call)
	} else {
		err = call(ctx)
	}
	if err != nil {
		return RouteGeometry{}, err
	}
	return geometry, nil
}

func (c *OSRMClient) routeURL(waypoints []LatLng, includeSteps bool) string {
	steps := "false"
	if includeSteps {
		steps = "true"
	}
	return c.baseURL + "/route/v1/driving/" + formatOSRMCoordinates(waypoints) +
		"?overview=full&geometries=polyline&steps=" + steps
}

type osrmTablePostRequest struct {
	// Coordinates use OSRM [lng, lat] ordering.
	Coordinates  [][2]float64 `json:"coordinates"`
	Sources      []int        `json:"sources"`
	Destinations []int        `json:"destinations"`
	Annotations  string       `json:"annotations"`
}

type osrmTableBlock struct {
	rowStart int
	rowEnd   int
	colStart int
	colEnd   int
}

func (c *OSRMClient) effectiveChunkCoords() int {
	if c.tableChunkCoords > 0 {
		return c.tableChunkCoords
	}
	return osrmDefaultChunkCoords
}

func (c *OSRMClient) effectiveSingleLimit() int {
	if c.tableChunkCoords > 0 {
		return c.tableChunkCoords
	}
	return osrmMaxSingleTableCoords
}

func (c *OSRMClient) effectiveMaxTableBytes() int {
	if c.maxTableBodyBytes > 0 && c.maxTableBodyBytes < (4<<20) {
		return c.maxTableBodyBytes
	}
	return osrmMaxTablePayloadBytes
}

func (c *OSRMClient) tableHTTPClient() *http.Client {
	if c.http == nil {
		return &http.Client{Timeout: osrmTableTimeout}
	}
	clone := *c.http
	if clone.Timeout == 0 || clone.Timeout < osrmTableTimeout {
		clone.Timeout = osrmTableTimeout
	}
	return &clone
}

// DistanceMatrix calls OSRM /table via chunked HTTP POST for pairwise driving distances in meters.
// Each chunk stays strictly under 4 MiB and bounded in coordinate count; returned matrix is NxN integers (meters).
func (c *OSRMClient) DistanceMatrix(ctx context.Context, points []LatLng) ([][]int, error) {
	if c == nil {
		return nil, fmt.Errorf("osrm: nil client")
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("osrm: empty points")
	}
	if len(points) == 1 {
		return [][]int{{0}}, nil
	}

	var matrix [][]int
	call := func(ctx context.Context) error {
		blocks := buildTableBlocks(len(points), c.effectiveChunkCoords(), c.effectiveSingleLimit())
		httpClient := c.tableHTTPClient()
		maxBody := c.effectiveMaxTableBytes()

		type blockResult struct {
			block osrmTableBlock
			sub   [][]int
		}
		results := make([]blockResult, len(blocks))

		workers := osrmTableWorkerLimit
		if len(blocks) < workers {
			workers = len(blocks)
		}

		subCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		jobs := make(chan int, len(blocks))
		for i := range blocks {
			jobs <- i
		}
		close(jobs)

		var errOnce sync.Once
		var firstErr error
		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()
				for idx := range jobs {
					if subCtx.Err() != nil {
						return
					}
					blk := blocks[idx]
					sub, err := c.fetchTableBlock(subCtx, httpClient, points, blk, maxBody)
					if err != nil {
						errOnce.Do(func() {
							firstErr = err
							cancel()
						})
						return
					}
					results[idx] = blockResult{block: blk, sub: sub}
				}
			}()
		}
		wg.Wait()
		if firstErr != nil {
			return firstErr
		}

		out := make([][]int, len(points))
		for i := range out {
			out[i] = make([]int, len(points))
		}
		for _, r := range results {
			for i, row := range r.sub {
				copy(out[r.block.rowStart+i][r.block.colStart:r.block.colEnd], row)
			}
		}
		matrix = out
		return nil
	}

	var err error
	if c.breaker != nil {
		err = c.breaker.Do(ctx, call)
	} else {
		err = call(ctx)
	}
	if err != nil {
		return nil, err
	}
	return matrix, nil
}

func buildTableBlocks(n, chunkCoords, singleLimit int) []osrmTableBlock {
	if n <= singleLimit {
		return []osrmTableBlock{{rowStart: 0, rowEnd: n, colStart: 0, colEnd: n}}
	}
	var blocks []osrmTableBlock
	for r := 0; r < n; r += chunkCoords {
		rEnd := r + chunkCoords
		if rEnd > n {
			rEnd = n
		}
		for c := 0; c < n; c += chunkCoords {
			cEnd := c + chunkCoords
			if cEnd > n {
				cEnd = n
			}
			blocks = append(blocks, osrmTableBlock{
				rowStart: r,
				rowEnd:   rEnd,
				colStart: c,
				colEnd:   cEnd,
			})
		}
	}
	return blocks
}

func buildTablePostPayload(points []LatLng, blk osrmTableBlock) ([]byte, error) {
	rows := blk.rowEnd - blk.rowStart
	cols := blk.colEnd - blk.colStart
	var coords [][2]float64
	sources := make([]int, rows)
	destinations := make([]int, cols)

	if blk.rowStart == blk.colStart && blk.rowEnd == blk.colEnd {
		coords = make([][2]float64, rows)
		for i := 0; i < rows; i++ {
			p := points[blk.rowStart+i]
			coords[i] = [2]float64{p.Lng, p.Lat}
			sources[i] = i
			destinations[i] = i
		}
	} else {
		coords = make([][2]float64, rows+cols)
		for i := 0; i < rows; i++ {
			p := points[blk.rowStart+i]
			coords[i] = [2]float64{p.Lng, p.Lat}
			sources[i] = i
		}
		for j := 0; j < cols; j++ {
			p := points[blk.colStart+j]
			coords[rows+j] = [2]float64{p.Lng, p.Lat}
			destinations[j] = rows + j
		}
	}

	return json.Marshal(osrmTablePostRequest{
		Coordinates:  coords,
		Sources:      sources,
		Destinations: destinations,
		Annotations:  "distance",
	})
}

func (c *OSRMClient) fetchTableBlock(ctx context.Context, httpClient *http.Client, points []LatLng, blk osrmTableBlock, maxBodyBytes int) ([][]int, error) {
	bodyBytes, err := buildTablePostPayload(points, blk)
	if err != nil {
		return nil, fmt.Errorf("osrm table marshal: %w", err)
	}
	rows := blk.rowEnd - blk.rowStart
	cols := blk.colEnd - blk.colStart
	if len(bodyBytes) >= maxBodyBytes {
		// Recursively bisect the larger dimension until serialized request payload fits < 4 MiB.
		if rows >= cols && rows > 1 {
			mid := blk.rowStart + rows/2
			top, err := c.fetchTableBlock(ctx, httpClient, points, osrmTableBlock{rowStart: blk.rowStart, rowEnd: mid, colStart: blk.colStart, colEnd: blk.colEnd}, maxBodyBytes)
			if err != nil {
				return nil, err
			}
			bot, err := c.fetchTableBlock(ctx, httpClient, points, osrmTableBlock{rowStart: mid, rowEnd: blk.rowEnd, colStart: blk.colStart, colEnd: blk.colEnd}, maxBodyBytes)
			if err != nil {
				return nil, err
			}
			return append(top, bot...), nil
		}
		if cols > 1 {
			mid := blk.colStart + cols/2
			left, err := c.fetchTableBlock(ctx, httpClient, points, osrmTableBlock{rowStart: blk.rowStart, rowEnd: blk.rowEnd, colStart: blk.colStart, colEnd: mid}, maxBodyBytes)
			if err != nil {
				return nil, err
			}
			right, err := c.fetchTableBlock(ctx, httpClient, points, osrmTableBlock{rowStart: blk.rowStart, rowEnd: blk.rowEnd, colStart: mid, colEnd: blk.colEnd}, maxBodyBytes)
			if err != nil {
				return nil, err
			}
			merged := make([][]int, rows)
			for i := 0; i < rows; i++ {
				merged[i] = append(left[i], right[i]...)
			}
			return merged, nil
		}
		return nil, fmt.Errorf("osrm table chunk request %d bytes exceeds max %d", len(bodyBytes), maxBodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tablePostURL(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("osrm table request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osrm table http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, (4<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("osrm table read: %w", err)
	}
	if len(respBody) >= (4 << 20) {
		return nil, fmt.Errorf("osrm table response exceeds 4 MiB cap")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osrm table status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var payload osrmTableResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("osrm table decode: %w", err)
	}
	if !strings.EqualFold(payload.Code, "Ok") {
		return nil, fmt.Errorf("osrm table code %q", payload.Code)
	}
	if len(payload.Distances) != rows {
		return nil, fmt.Errorf("osrm table size %d != %d", len(payload.Distances), rows)
	}
	sub := make([][]int, rows)
	for i := 0; i < rows; i++ {
		if len(payload.Distances[i]) != cols {
			return nil, fmt.Errorf("osrm table row %d size %d != %d", i, len(payload.Distances[i]), cols)
		}
		sub[i] = make([]int, cols)
		for j := 0; j < cols; j++ {
			d := payload.Distances[i][j]
			if d < 0 || math.IsNaN(d) {
				// OSRM uses null/-1 for unreachable; leave 0 and let caller fill haversine.
				sub[i][j] = 0
				continue
			}
			sub[i][j] = int(d)
		}
	}
	return sub, nil
}

func (c *OSRMClient) tablePostURL() string {
	return c.baseURL + "/table/v1/driving/?annotations=distance"
}

func formatOSRMCoordinates(waypoints []LatLng) string {
	var b strings.Builder
	for i, point := range waypoints {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(strconv.FormatFloat(point.Lng, 'f', 6, 64))
		b.WriteByte(',')
		b.WriteString(strconv.FormatFloat(point.Lat, 'f', 6, 64))
	}
	return b.String()
}
