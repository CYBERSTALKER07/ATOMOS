import re

with open("apps/backend-go/events/types.go", "r") as f:
    content = f.read()

content = content.replace('GPSLng                float64 `json:"gps_lng,omitempty"`\n}', 'GPSLng                float64 `json:"gps_lng,omitempty"`\n\tMessage               string  `json:"message,omitempty"`\n}')

with open("apps/backend-go/events/types.go", "w") as f:
    f.write(content)
