// Package routing — live ETA computation using Google Maps Directions API.
// Called on driver depart (traffic-aware) and after each delivery completion
// to refresh remaining stop ETAs from the driver's current position.
package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// ComputeLiveETAs calls Google Maps Directions API with departure_time=now for
// traffic-aware ETAs. It writes EstimatedArrivalAt, EstimatedDurationSec, and
// EstimatedDistanceM per remaining order, plus EstimatedReturnAt and
// ReturnDurationSec on the Drivers row.
//
// originLat/originLng: driver's current position (or depot on initial depart).
// depotLat/depotLng: supplier's warehouse for the return leg.
// departedAt: the timestamp the driver physically departed (used as the ETA base).
func ComputeLiveETAs(
	ctx context.Context,
	client *spanner.Client,
	apiKey string,
	originLat, originLng float64,
	depotLat, depotLng float64,
	driverID string,
	departedAt time.Time,
) error {
	// 1. Fetch remaining orders in sequence
	orders, err := GetRemainingOrders(ctx, client, driverID)
	if err != nil {
		return fmt.Errorf("eta: failed to fetch remaining orders: %w", err)
	}
	if len(orders) == 0 {
		log.Printf("[ETA] No remaining orders for driver %s — computing return only", driverID)
		return computeReturnOnlyETA(ctx, client, apiKey, originLat, originLng, depotLat, depotLng, driverID, departedAt)
	}

	origin := fmt.Sprintf("%f,%f", originLat, originLng)
	depot := fmt.Sprintf("%f,%f", depotLat, depotLng)

	// 2. Build waypoints for remaining stops → depot
	waypoints := make([]string, 0, len(orders))
	for _, o := range orders {
		waypoints = append(waypoints, fmt.Sprintf("%f,%f", o.ParsedLat, o.ParsedLng))
	}

	// 3. Call Directions API with departure_time=now for traffic
	reqURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json"+
			"?origin=%s&destination=%s"+
			"&waypoints=%s"+
			"&departure_time=now"+
			"&key=%s",
		url.QueryEscape(origin),
		url.QueryEscape(depot), // destination = warehouse (round trip)
		url.QueryEscape(strings.Join(waypoints, "|")),
		apiKey,
	)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return fmt.Errorf("eta: Google Maps request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("eta: Google Maps returned HTTP %d", resp.StatusCode)
	}

	var result mapsDirectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("eta: failed to decode Maps response: %w", err)
	}

	if result.Status != "OK" || len(result.Routes) == 0 {
		return fmt.Errorf("eta: Maps rejected the request (status=%q)", result.Status)
	}

	legs := result.Routes[0].Legs
	// legs[0] = origin → stop1, legs[1] = stop1 → stop2, ..., legs[N] = stopN → depot

	// 4. Build mutations with cumulative ETAs
	mutations := make([]*spanner.Mutation, 0, len(orders)+1)
	cumulativeSec := 0

	for i, o := range orders {
		if i < len(legs) {
			cumulativeSec += legs[i].Duration.Value
		}
		arrivalAt := departedAt.Add(time.Duration(cumulativeSec) * time.Second)

		mutations = append(mutations, spanner.Update("Orders",
			[]string{"OrderId", "EstimatedArrivalAt", "EstimatedDurationSec", "EstimatedDistanceM"},
			[]interface{}{o.OrderID, arrivalAt, int64(cumulativeSec), int64(safeGetLegDistance(legs, i))},
		))
	}

	// 5. Return leg: last stop → depot
	returnSec := 0
	if len(legs) > len(orders) {
		returnSec = legs[len(orders)].Duration.Value
	}
	totalSec := cumulativeSec + returnSec
	returnAt := departedAt.Add(time.Duration(totalSec) * time.Second)

	mutations = append(mutations, spanner.Update("Drivers",
		[]string{"DriverId", "EstimatedReturnAt", "ReturnDurationSec"},
		[]interface{}{driverID, returnAt, int64(returnSec)},
	))

	// 6. Commit
	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	}); err != nil {
		return fmt.Errorf("eta: Spanner commit failed: %w", err)
	}

	log.Printf("[ETA] Updated %d stop ETAs + return for driver %s (total route: %d sec)", len(orders), driverID, totalSec)
	return nil
}

// ComputeReturnETA calculates only the return-to-warehouse ETA using Google Maps
// Distance Matrix API. Called after the last delivery is completed.
func ComputeReturnETA(
	ctx context.Context,
	client *spanner.Client,
	apiKey string,
	driverLat, driverLng float64,
	depotLat, depotLng float64,
	driverID string,
) (returnAt time.Time, durationSec int64, err error) {
	origin := fmt.Sprintf("%f,%f", driverLat, driverLng)
	dest := fmt.Sprintf("%f,%f", depotLat, depotLng)

	reqURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/distancematrix/json"+
			"?origins=%s&destinations=%s"+
			"&departure_time=now"+
			"&key=%s",
		url.QueryEscape(origin),
		url.QueryEscape(dest),
		apiKey,
	)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("eta: Distance Matrix request failed: %w", err)
	}
	defer resp.Body.Close()

	var dmResp distanceMatrixResponse
	if err := json.NewDecoder(resp.Body).Decode(&dmResp); err != nil {
		return time.Time{}, 0, fmt.Errorf("eta: failed to decode Distance Matrix response: %w", err)
	}

	if dmResp.Status != "OK" || len(dmResp.Rows) == 0 || len(dmResp.Rows[0].Elements) == 0 {
		return time.Time{}, 0, fmt.Errorf("eta: Distance Matrix returned status=%q", dmResp.Status)
	}

	elem := dmResp.Rows[0].Elements[0]
	if elem.Status != "OK" {
		return time.Time{}, 0, fmt.Errorf("eta: element status=%q", elem.Status)
	}

	now := time.Now().UTC()
	durSec := int64(elem.Duration.Value)
	returnAt = now.Add(time.Duration(durSec) * time.Second)

	// Write to Spanner
	_, applyErr := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Drivers",
				[]string{"DriverId", "EstimatedReturnAt", "ReturnDurationSec"},
				[]interface{}{driverID, returnAt, durSec},
			),
		})
	})
	if applyErr != nil {
		return returnAt, durSec, fmt.Errorf("eta: Spanner write failed: %w", applyErr)
	}

	log.Printf("[ETA] Return ETA for driver %s: %d sec, arrival at %s", driverID, durSec, returnAt.Format(time.RFC3339))
	return returnAt, durSec, nil
}

// computeReturnOnlyETA is used when there are no remaining orders but we need
// the return leg (all deliveries completed, driver heading back).
func computeReturnOnlyETA(
	ctx context.Context,
	client *spanner.Client,
	apiKey string,
	driverLat, driverLng float64,
	depotLat, depotLng float64,
	driverID string,
	departedAt time.Time,
) error {
	_, _, err := ComputeReturnETA(ctx, client, apiKey, driverLat, driverLng, depotLat, depotLng, driverID)
	return err
}

// GetRemainingOrders fetches orders in IN_TRANSIT or LOADED state for a driver,
// ordered by SequenceIndex. Used for live ETA recalculation.
func GetRemainingOrders(ctx context.Context, client *spanner.Client, driverID string) ([]DeliveryOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, DriverId, ShopLocation
		      FROM Orders
		      WHERE DriverId = @driverID AND State IN ('IN_TRANSIT', 'LOADED')
		      ORDER BY SequenceIndex ASC NULLS LAST`,
		Params: map[string]interface{}{
			"driverID": driverID,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var orders []DeliveryOrder
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var o DeliveryOrder
		var shopLoc spanner.NullString
		if err := row.Columns(&o.OrderID, &o.DriverID, &shopLoc); err != nil {
			return nil, fmt.Errorf("eta: row parse failed: %w", err)
		}
		if shopLoc.Valid {
			o.ShopLocation = shopLoc.StringVal
			lat, lng, parseErr := parseWKTPoint(shopLoc.StringVal)
			if parseErr != nil {
				log.Printf("[ETA] WARN: could not parse ShopLocation for order %s: %v", o.OrderID, parseErr)
				continue
			}
			o.ParsedLat = lat
			o.ParsedLng = lng
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// ── Distance Matrix API Types ─────────────────────────────────────────────

type distanceMatrixResponse struct {
	Status string              `json:"status"`
	Rows   []distanceMatrixRow `json:"rows"`
}

type distanceMatrixRow struct {
	Elements []distanceMatrixElement `json:"elements"`
}

type distanceMatrixElement struct {
	Status   string    `json:"status"`
	Duration mapsValue `json:"duration"`
	Distance mapsValue `json:"distance"`
}
