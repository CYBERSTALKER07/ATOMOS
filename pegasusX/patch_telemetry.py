import re

with open("apps/backend-go/telemetryroutes/routes.go", "r") as f:
    content = f.read()

append_str = """
type SensorUpdate struct {
	DeviceId string   `json:"device_id"`
	Humidity *float64 `json:"humidity,omitempty"`
	ShockG   *float64 `json:"shock_g,omitempty"`
	Tilt     *float64 `json:"tilt,omitempty"`
	Tampered *bool    `json:"tampered,omitempty"`
}

func HandleSensors(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeTelemetryJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		identity, err := d.resolveDriverIdentity(r)
		if err != nil {
			d.writeIdentityError(w, err)
			return
		}
		var update SensorUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeTelemetryJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		
		alert := false
		if update.Tampered != nil && *update.Tampered {
			alert = true
		}
		if update.ShockG != nil && *update.ShockG > 5.0 {
			alert = true
		}
		
		if alert && d.Log != nil {
			d.Log.Warn("sensor threshold alert triggered", "device", update.DeviceId, "driver", identity.DriverID)
		}
		
		if d.TelemetryHub != nil {
			payload, _ := json.Marshal(map[string]any{
				"type": "SENSOR_ALERT",
				"device_id": update.DeviceId,
				"driver_id": identity.DriverID,
				"tampered": update.Tampered,
				"shock_g": update.ShockG,
			})
			d.TelemetryHub.Broadcast(r.Context(), "telemetry:supplier:"+identity.SupplierID, payload)
		}
		
		writeTelemetryJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
	}
}
"""

content += append_str
content = content.replace('rr.Post("/v1/telemetry/driver/location", HandleDriverLocation(d))', 'rr.Post("/v1/telemetry/driver/location", HandleDriverLocation(d))\n\t\trr.Post("/v1/telemetry/sensors", HandleSensors(d))')

with open("apps/backend-go/telemetryroutes/routes.go", "w") as f:
    f.write(content)
