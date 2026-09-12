import re

with open('apps/backend-go/dispatch/optimizerclient/client.go', 'r') as f:
    content = f.read()

# Fix 1: Matrix offset bug
old_matrix = """func buildDistanceMatrix(ctx context.Context, osrm *routing.OSRMClient, vehicles []contract.Vehicle, stops []contract.Stop) [][]int {
	points := make([]routing.LatLng, 0, len(vehicles)*2+len(stops))
	for _, v := range vehicles {
		points = append(points, routing.LatLng{Lat: v.StartLat, Lng: v.StartLng})
		if v.EndLat != 0 || v.EndLng != 0 {
			if absFloat(v.EndLat-v.StartLat) > 1e-9 || absFloat(v.EndLng-v.StartLng) > 1e-9 {
				points = append(points, routing.LatLng{Lat: v.EndLat, Lng: v.EndLng})
			}
		}
	}
	for _, s := range stops {
		points = append(points, routing.LatLng{Lat: s.Lat, Lng: s.Lng})
	}"""

new_matrix = """func buildDistanceMatrix(ctx context.Context, osrm *routing.OSRMClient, vehicles []contract.Vehicle, stops []contract.Stop) [][]int {
	// Strict contract alignment: Exactly 1 node per vehicle (Start), then 1 node per Stop.
	// Distinct End nodes must NOT be interleaved here as they would shift the Stop indices
	// which ai-worker hardcodes to len(Vehicles)+j.
	points := make([]routing.LatLng, 0, len(vehicles)+len(stops))
	for _, v := range vehicles {
		points = append(points, routing.LatLng{Lat: v.StartLat, Lng: v.StartLng})
	}
	for _, s := range stops {
		points = append(points, routing.LatLng{Lat: s.Lat, Lng: s.Lng})
	}"""

if old_matrix in content:
    content = content.replace(old_matrix, new_matrix)
    print("Patched matrix bug.")
else:
    print("Failed to patch matrix bug.")

with open('apps/backend-go/dispatch/optimizerclient/client.go', 'w') as f:
    f.write(content)
