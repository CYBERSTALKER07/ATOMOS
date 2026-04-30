// Ghost Driver — Simulates a truck driving down Amir Temur Avenue, Tashkent.
// Fires GPS pings into the WebSocket fleet hub every 2 seconds.
// Usage: go run cmd/ghost-driver/main.go
package main

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type LocationUpdate struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Waypoints along Amir Temur Avenue → Chorsu Bazaar, Tashkent
var route = []LocationUpdate{
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2995, Longitude: 69.2401},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2980, Longitude: 69.2412},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2965, Longitude: 69.2425},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2950, Longitude: 69.2438},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2935, Longitude: 69.2451},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2920, Longitude: 69.2465},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2908, Longitude: 69.2480},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2895, Longitude: 69.2495},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2880, Longitude: 69.2510},
	{DriverID: "TRUCK-TASH-01", Latitude: 41.2866, Longitude: 69.2525}, // Chorsu
}

func main() {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/fleet"}
	log.Printf("[GHOST DRIVER] Dialing: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("[GHOST DRIVER] Connection failed: %v", err)
	}
	defer conn.Close()
	log.Printf("[GHOST DRIVER] Pipe open. Starting route simulation for TRUCK-TASH-01...")

	for i, waypoint := range route {
		payload, _ := json.Marshal(waypoint)
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("[GHOST DRIVER] Send failed at waypoint %d: %v", i, err)
			break
		}
		log.Printf("[GHOST DRIVER] Ping %d/10 → Lat: %.4f, Lng: %.4f", i+1, waypoint.Latitude, waypoint.Longitude)
		time.Sleep(2 * time.Second)
	}

	log.Printf("[GHOST DRIVER] Route complete. TRUCK-TASH-01 has arrived at Chorsu Bazaar.")
}
