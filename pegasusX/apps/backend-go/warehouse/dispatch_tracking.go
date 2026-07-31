package warehouse

import (
	"net/http"
	"strings"
)

type PartnerFilterMetric struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type VehicleShipmentCard struct {
	ID                string   `json:"id"`
	Code              string   `json:"code"`
	Status            string   `json:"status"` // WAITING, ON_ROUTE, COMPLETED, DELAYED
	VehicleType       string   `json:"vehicle_type"` // VAN, SEMI_TRUCK, BOX_TRUCK, FLATBED
	ETASeconds        int      `json:"eta_seconds"`
	DistanceMilesLeft int      `json:"distance_miles_left"`
	StopsCount        int      `json:"stops_count"`
	StopsSummary      []string `json:"stops_summary"`
	DriverName        string   `json:"driver_name,omitempty"`
	DriverPhone       string   `json:"driver_phone,omitempty"`
	PartnerID         string   `json:"partner_id,omitempty"`
	PartnerName       string   `json:"partner_name,omitempty"`
}

type FleetDispatchOverviewResponse struct {
	TotalCount     int                   `json:"total_count"`
	ActiveCount    int                   `json:"active_count"`
	InactiveCount  int                   `json:"inactive_count"`
	PartnerFilters []PartnerFilterMetric `json:"partner_filters"`
	Shipments      []VehicleShipmentCard `json:"shipments"`
}

// HandleDispatchTracking serves GET /v1/warehouse/dispatch/tracking.
func (s *Service) HandleDispatchTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	partnerFilter := strings.TrimSpace(r.URL.Query().Get("partner_id"))
	statusFilter := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("status_filter")))

	partnerMetrics := []PartnerFilterMetric{
		{ID: "p1", Name: "Lockman", Count: 24},
		{ID: "p2", Name: "Mertz LLC", Count: 22},
		{ID: "p3", Name: "Corkery", Count: 8},
		{ID: "p4", Name: "Kuhn and Sons", Count: 5},
		{ID: "p5", Name: "Weissnat and Sons", Count: 3},
		{ID: "p6", Name: "Morissette Inc", Count: 2},
		{ID: "p7", Name: "Deckow LLC", Count: 2},
	}

	allShipments := []VehicleShipmentCard{
		{
			ID:                "v-936383762",
			Code:              "XR-936383762",
			Status:            "WAITING",
			VehicleType:       "VAN",
			ETASeconds:        13933,
			DistanceMilesLeft: 36,
			StopsCount:        4,
			StopsSummary:      []string{"61115 Claudio Walks", "15303 Bohringer Inlet", "457 Saint Veda", "3516 Ryan Valleys"},
			DriverName:        "Alex Mercer",
			DriverPhone:       "+1 555-0142",
			PartnerID:         "p1",
			PartnerName:       "Lockman",
		},
		{
			ID:                "v-113949207",
			Code:              "AL-113949207",
			Status:            "WAITING",
			VehicleType:       "VAN",
			ETASeconds:        8233,
			DistanceMilesLeft: 64,
			StopsCount:        5,
			StopsSummary:      []string{"42047 Verta Ridge", "22920 Shondra Street", "6722 Locascio Mount", "0732 Allen Crossing", "33900 Shonel Street"},
			DriverName:        "Marcus Vance",
			DriverPhone:       "+1 555-0189",
			PartnerID:         "p2",
			PartnerName:       "Mertz LLC",
		},
		{
			ID:                "v-118945307",
			Code:              "AL-118945307",
			Status:            "ON_ROUTE",
			VehicleType:       "VAN",
			ETASeconds:        5596,
			DistanceMilesLeft: 90,
			StopsCount:        5,
			StopsSummary:      []string{"0298 Hermann Corners", "50578 Schuppe Streamway", "88364 Edison Valley", "412 Elenor Way", "4074 Hegmann Pike"},
			DriverName:        "Victor Brooks",
			DriverPhone:       "+1 555-0211",
			PartnerID:         "p3",
			PartnerName:       "Corkery",
		},
		{
			ID:                "v-752069247",
			Code:              "SD-752069247",
			Status:            "ON_ROUTE",
			VehicleType:       "SEMI_TRUCK",
			ETASeconds:        5035,
			DistanceMilesLeft: 38,
			StopsCount:        5,
			StopsSummary:      []string{"2821 Keelie Hills", "36716 Audreanne Date", "399 Lorine Island", "0732 Allen Crossing", "4164 Torrance Plaza"},
			DriverName:        "John Miller",
			DriverPhone:       "+1 555-0199",
			PartnerID:         "p1",
			PartnerName:       "Lockman",
		},
		{
			ID:                "v-752263347",
			Code:              "SD-752263347",
			Status:            "ON_ROUTE",
			VehicleType:       "SEMI_TRUCK",
			ETASeconds:        2596,
			DistanceMilesLeft: 98,
			StopsCount:        4,
			StopsSummary:      []string{"3259 Haley Wells", "34430 Mraz Locks", "50520 Beatty Burg", "61115 Claudio Walks"},
			DriverName:        "David Ray",
			DriverPhone:       "+1 555-0304",
			PartnerID:         "p4",
			PartnerName:       "Kuhn and Sons",
		},
		{
			ID:                "v-916427621",
			Code:              "XR-916427621",
			Status:            "ON_ROUTE",
			VehicleType:       "SEMI_TRUCK",
			ETASeconds:        1456,
			DistanceMilesLeft: 112,
			StopsCount:        3,
			StopsSummary:      []string{"62597 Viviane Harbors", "0732 Allen Crossing", "9667 Huel Drive"},
			DriverName:        "Sam West",
			DriverPhone:       "+1 555-0455",
			PartnerID:         "p5",
			PartnerName:       "Weissnat and Sons",
		},
	}

	filtered := make([]VehicleShipmentCard, 0, len(allShipments))
	for _, s := range allShipments {
		if partnerFilter != "" && s.PartnerID != partnerFilter {
			continue
		}
		if statusFilter == "ACTIVE" && s.Status != "ON_ROUTE" {
			continue
		}
		if statusFilter == "INACTIVE" && s.Status != "WAITING" {
			continue
		}
		filtered = append(filtered, s)
	}

	writeJSON(w, http.StatusOK, FleetDispatchOverviewResponse{
		TotalCount:     71,
		ActiveCount:    34,
		InactiveCount:  37,
		PartnerFilters: partnerMetrics,
		Shipments:      filtered,
	})
}
