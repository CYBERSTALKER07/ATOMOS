import re

with open("apps/backend-go/telemetry/location_store.go", "r") as f:
    content = f.read()

pattern = re.compile(r'type DriverLocation struct \{\n\tDriverID\s+string\s+`json:"driver_id"`\n\tSupplierID\s+string\s+`json:"supplier_id"`\n\tLat\s+float64\s+`json:"lat"`\n\tLng\s+float64\s+`json:"lng"`\n\tLatitude\s+float64\s+`json:"latitude"`\n\tLongitude\s+float64\s+`json:"longitude"`\n\tVelocity\s+\*float64\s+`json:"velocity,omitempty"`\n\tHeading\s+\*float64\s+`json:"heading,omitempty"`\n\tReportedAt\s+time\.Time\s+`json:"reported_at"`\n\tReceivedAt\s+time\.Time\s+`json:"received_at"`\n\tStaleAfterSeconds int\s+`json:"stale_after_seconds"`\n\}')

replacement = r"""type DriverLocation struct {
	DriverID          string    `json:"driver_id"`
	SupplierID        string    `json:"supplier_id"`
	Lat               float64   `json:"lat"`
	Lng               float64   `json:"lng"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	Velocity          *float64  `json:"velocity,omitempty"`
	Heading           *float64  `json:"heading,omitempty"`
	Humidity          *float64  `json:"humidity,omitempty"`
	ShockG            *float64  `json:"shock_g,omitempty"`
	Tilt              *float64  `json:"tilt,omitempty"`
	Tampered          *bool     `json:"tampered,omitempty"`
	ReportedAt        time.Time `json:"reported_at"`
	ReceivedAt        time.Time `json:"received_at"`
	StaleAfterSeconds int       `json:"stale_after_seconds"`
}"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/telemetry/location_store.go", "w") as f:
    f.write(content)

with open("apps/backend-go/telemetryroutes/routes.go", "r") as f:
    content = f.read()

pattern2 = re.compile(r'type driverLocationPayload struct \{\n\tDriverID\s+string\s+`json:"driver_id"`\n\tSupplierID\s+string\s+`json:"supplier_id"`\n\tLat\s+float64\s+`json:"lat"`\n\tLng\s+float64\s+`json:"lng"`\n\tLatitude\s+float64\s+`json:"latitude"`\n\tLongitude\s+float64\s+`json:"longitude"`\n\tVelocity\s+\*float64\s+`json:"velocity,omitempty"`\n\tHeading\s+\*float64\s+`json:"heading,omitempty"`\n\tReportedAt\s+string\s+`json:"reported_at"`\n\tReceivedAt\s+string\s+`json:"received_at"`\n\tStaleAfterSeconds int\s+`json:"stale_after_seconds"`\n\}')

replacement2 = r"""type driverLocationPayload struct {
	DriverID          string   `json:"driver_id"`
	SupplierID        string   `json:"supplier_id"`
	Lat               float64  `json:"lat"`
	Lng               float64  `json:"lng"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	Velocity          *float64 `json:"velocity,omitempty"`
	Heading           *float64 `json:"heading,omitempty"`
	Humidity          *float64 `json:"humidity,omitempty"`
	ShockG            *float64 `json:"shock_g,omitempty"`
	Tilt              *float64 `json:"tilt,omitempty"`
	Tampered          *bool    `json:"tampered,omitempty"`
	ReportedAt        string   `json:"reported_at"`
	ReceivedAt        string   `json:"received_at"`
	StaleAfterSeconds int      `json:"stale_after_seconds"`
}"""
content = pattern2.sub(replacement2, content)

with open("apps/backend-go/telemetryroutes/routes.go", "w") as f:
    f.write(content)
