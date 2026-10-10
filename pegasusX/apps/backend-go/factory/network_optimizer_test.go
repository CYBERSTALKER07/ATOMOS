package factory

import "testing"

func TestSelectOptimalFactory_SPEEDPicksShorterTransit(t *testing.T) {
	lanes := []SupplyLane{
		{FactoryID: "far", DampenedTransitHours: 48, FreightCostMinor: 1, CarbonScoreKg: 1, IsActive: true},
		{FactoryID: "near", DampenedTransitHours: 8, FreightCostMinor: 999, CarbonScoreKg: 50, IsActive: true},
	}
	got := SelectOptimalFactoryFromLanes(NetworkModeSpeed, lanes)
	if got.FactoryID != "near" {
		t.Fatalf("SPEED factory=%q want near", got.FactoryID)
	}
	if got.CapacityLabel != "UNLIMITED" {
		t.Fatalf("capacity observer=%q want UNLIMITED", got.CapacityLabel)
	}
}

func TestSelectOptimalFactory_ECONOMYPicksCheaper(t *testing.T) {
	lanes := []SupplyLane{
		{FactoryID: "cheap", DampenedTransitHours: 40, FreightCostMinor: 10, IsActive: true},
		{FactoryID: "fast", DampenedTransitHours: 4, FreightCostMinor: 9000, IsActive: true},
	}
	got := SelectOptimalFactoryFromLanes(NetworkModeEconomy, lanes)
	if got.FactoryID != "cheap" {
		t.Fatalf("ECONOMY factory=%q want cheap", got.FactoryID)
	}
}

func TestSelectOptimalFactory_LOWCARBON(t *testing.T) {
	lanes := []SupplyLane{
		{FactoryID: "green", CarbonScoreKg: 1, DampenedTransitHours: 40, IsActive: true},
		{FactoryID: "soot", CarbonScoreKg: 80, DampenedTransitHours: 4, IsActive: true},
	}
	got := SelectOptimalFactoryFromLanes(NetworkModeLowCarbon, lanes)
	if got.FactoryID != "green" {
		t.Fatalf("LOW_CARBON factory=%q want green", got.FactoryID)
	}
}

func TestSelectOptimalFactory_BALANCEDPrefersWeightedCombo(t *testing.T) {
	lanes := []SupplyLane{
		{FactoryID: "costly-fast", DampenedTransitHours: 4, FreightCostMinor: 20000, CarbonScoreKg: 80, IsActive: true},
		{FactoryID: "cheap-slow", DampenedTransitHours: 10, FreightCostMinor: 10, CarbonScoreKg: 1, IsActive: true},
	}
	got := SelectOptimalFactoryFromLanes(NetworkModeBalanced, lanes)
	if got.FactoryID != "cheap-slow" {
		t.Fatalf("BALANCED factory=%q want cheap-slow (0.5*transit + 0.0003*cost + 0.2*carbon)", got.FactoryID)
	}
}

func TestSelectOptimalFactory_MANUALONLYEmpty(t *testing.T) {
	lanes := []SupplyLane{
		{FactoryID: "a", DampenedTransitHours: 1, IsActive: true},
	}
	got := SelectOptimalFactoryFromLanes(NetworkModeManualOnly, lanes)
	if got.FactoryID != "" {
		t.Fatalf("MANUAL_ONLY factory=%q want empty", got.FactoryID)
	}
}

func TestSelectFallbackFactory_Nearest(t *testing.T) {
	cands := []FactoryCandidate{
		{FactoryID: "far", Lat: 42, Lng: 69, IsActive: true},
		{FactoryID: "near", Lat: 41.32, Lng: 69.25, IsActive: true},
	}
	got := SelectFallbackFactory(41.31, 69.24, "", "", cands)
	if got.FactoryID != "near" {
		t.Fatalf("fallback=%q want near", got.FactoryID)
	}
}
