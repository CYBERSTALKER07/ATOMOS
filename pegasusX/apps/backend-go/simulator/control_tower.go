package simulator

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/uber/h3-go/v4"
)

// NetworkNode represents a node in the Control Tower live EKG graph
type NetworkNode struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Label  string `json:"label"`
	Status string `json:"status"` // "active", "idle", "busy", "offline"
}

// NetworkLink represents a connection between two nodes
type NetworkLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  int    `json:"value"`
}

// ControlTowerNetworkPayload is the WS payload for the network graph
type ControlTowerNetworkPayload struct {
	Type  string        `json:"type"` // "control_tower_network"
	Nodes []NetworkNode `json:"nodes"`
	Links []NetworkLink `json:"links"`
}

// H3Density represents order/fleet density at a specific H3 index
type H3Density struct {
	Hex   string `json:"hex"`
	Count int    `json:"count"`
}

// ControlTowerH3Payload is the WS payload for the spatial map
type ControlTowerH3Payload struct {
	Type string      `json:"type"` // "control_tower_h3"
	Data []H3Density `json:"data"`
}

func StartControlTowerSimulation(telemetryHub *ws.Hub, supplierID string, warehouseID string) {
	if telemetryHub == nil {
		return
	}
	// Defense-in-depth: the random-data simulator is demo/dev-only. Bootstrap
	// already gates on CONTROL_TOWER_SIMULATOR_ENABLED + env; guard here too so
	// no future callsite can emit fabricated telemetry on real environments.
	if auth.IsSandbox() || auth.IsProduction() || auth.IsStaging() {
		return
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			ctx := context.Background()

			// 1. Generate Network Graph Data
			networkPayload := generateMockNetwork()
			networkBytes, _ := json.Marshal(networkPayload)

			// 2. Generate H3 Spatial Map Data
			h3Payload := generateMockH3Data()
			h3Bytes, _ := json.Marshal(h3Payload)

			// Emit to supplier room
			if supplierID != "" {
				telemetryHub.Broadcast(ctx, "telemetry:supplier:"+supplierID, networkBytes)
				telemetryHub.Broadcast(ctx, "telemetry:supplier:"+supplierID, h3Bytes)
			}

			// Emit to warehouse room (they might listen to the same telemetry channels or warehouse specific)
			// According to handler.go, warehouse admins subscribe to telemetry:supplier:<supplierID>
			// So broadcasting to supplier ID is sufficient for both Supplier and Warehouse portals in single-tenant mode.
		}
	}()
}

func generateMockNetwork() ControlTowerNetworkPayload {
	statuses := []string{"active", "idle", "busy"}
	return ControlTowerNetworkPayload{
		Type: "control_tower_network",
		Nodes: []NetworkNode{
			{ID: "WH-1", Type: "warehouse", Label: "Central Hub", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "WH-2", Type: "warehouse", Label: "East DC", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "RT-1", Type: "retailer", Label: "Store Alpha", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "RT-2", Type: "retailer", Label: "Store Beta", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "RT-3", Type: "retailer", Label: "Store Gamma", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "DR-1", Type: "driver", Label: "Driver 104", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "DR-2", Type: "driver", Label: "Driver 211", Status: statuses[rand.Intn(len(statuses))]},
			{ID: "DR-3", Type: "driver", Label: "Driver 305", Status: statuses[rand.Intn(len(statuses))]},
		},
		Links: []NetworkLink{
			{Source: "WH-1", Target: "RT-1", Value: rand.Intn(20) + 1},
			{Source: "WH-1", Target: "RT-2", Value: rand.Intn(10) + 1},
			{Source: "WH-2", Target: "RT-3", Value: rand.Intn(15) + 1},
			{Source: "WH-1", Target: "DR-1", Value: rand.Intn(5) + 1},
			{Source: "DR-1", Target: "RT-1", Value: rand.Intn(8) + 1},
			{Source: "WH-2", Target: "DR-2", Value: rand.Intn(6) + 1},
			{Source: "DR-2", Target: "RT-3", Value: rand.Intn(12) + 1},
			{Source: "WH-1", Target: "DR-3", Value: rand.Intn(4) + 1},
			{Source: "DR-3", Target: "RT-2", Value: rand.Intn(9) + 1},
		},
	}
}

func generateMockH3Data() ControlTowerH3Payload {
	data := make([]H3Density, 0, 50)
	centerLat := 37.74
	centerLng := -122.4

	// Create h3 cells
	latLng := h3.NewLatLng(centerLat, centerLng)
	centerCell, _ := h3.LatLngToCell(latLng, 8)

	// Get neighbors to form a cluster
	cells, _ := h3.GridDisk(centerCell, 4)

	for _, cell := range cells {
		// Only include some cells randomly to simulate sparse data
		if rand.Float32() > 0.3 {
			data = append(data, H3Density{
				Hex:   cell.String(),
				Count: rand.Intn(100),
			})
		}
	}

	return ControlTowerH3Payload{
		Type: "control_tower_h3",
		Data: data,
	}
}
