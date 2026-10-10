package dispatch

// ManifestChunk is one sub-manifest after SplitManifest.
type ManifestChunk struct {
	RouteID  string
	Orders   []GeoOrder
	VolumeVU float64
	Suffix   string
}

// ManifestGroup is SplitManifest output for one driver.
type ManifestGroup struct {
	DriverID    string
	TruckID     string
	TotalOrders int
	Chunks      []ManifestChunk
}
