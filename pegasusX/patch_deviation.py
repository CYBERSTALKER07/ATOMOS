import re

with open("apps/backend-go/routing/deviation.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func distancePointToSegmentMeters\(lat, lng float64, start, end LatLng\) float64 \{\n\tdx := end\.Lng - start\.Lng\n\tdy := end\.Lat - start\.Lat\n\tif dx == 0 && dy == 0 \{\n\t\treturn haversineMeters\(lat, lng, start\.Lat, start\.Lng\)\n\t\}\n\tt := \(\(lng-start\.Lng\)\*dx \+ \(lat-start\.Lat\)\*dy\) / \(dx\*dx \+ dy\*dy\)\n\tif t < 0 \{\n\t\tt = 0\n\t\} else if t > 1 \{\n\t\tt = 1\n\t\}\n\tclosestLat := start\.Lat \+ t\*dy\n\tclosestLng := start\.Lng \+ t\*dx\n\treturn haversineMeters\(lat, lng, closestLat, closestLng\)\n\}')

replacement = r"""func distancePointToSegmentMeters(lat, lng float64, start, end LatLng) float64 {
	avgLat := (start.Lat + end.Lat) / 2.0
	cosFactor := math.Cos(avgLat * math.Pi / 180.0)
	
	dx := (end.Lng - start.Lng) * cosFactor
	dy := end.Lat - start.Lat
	if dx == 0 && dy == 0 {
		return haversineMeters(lat, lng, start.Lat, start.Lng)
	}
	
	dlng := (lng - start.Lng) * cosFactor
	dlat := lat - start.Lat
	
	t := (dlng*dx + dlat*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closestLat := start.Lat + t*(end.Lat - start.Lat)
	closestLng := start.Lng + t*(end.Lng - start.Lng)
	return haversineMeters(lat, lng, closestLat, closestLng)
}"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/routing/deviation.go", "w") as f:
    f.write(content)
