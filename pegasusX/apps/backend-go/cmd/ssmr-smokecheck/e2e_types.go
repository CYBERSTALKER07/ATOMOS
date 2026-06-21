package main

// dispatchManifestHint carries warehouse dispatch execute output into the payloader journey.
type dispatchManifestHint struct {
	ManifestID string
	DriverID   string
	VehicleID  string
	OrderIDs   []string
}
