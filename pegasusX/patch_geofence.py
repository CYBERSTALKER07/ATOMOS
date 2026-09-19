import re

with open("apps/backend-go/order/service.go", "r") as f:
    content = f.read()

pattern = re.compile(r'\t\t\tif !req\.BypassGeofence && \(req\.Latitude != 0 \|\| req\.Longitude != 0\) \{\n\t\t\t\tcomputedDistance, err := validateOptionalGeofence\(ctx, req\.Latitude, req\.Longitude, orderRecord\)\n\t\t\t\tif err == nil \{\n\t\t\t\t\tdistanceM = computedDistance\n\t\t\t\t\}\n\t\t\t\treturn err\n\t\t\t\}')

replacement = r"""			if !req.BypassGeofence {
				if req.Latitude == 0 && req.Longitude == 0 {
					return errors.New("gps coordinates required for delivery submission")
				}
				computedDistance, err := validateOptionalGeofence(ctx, req.Latitude, req.Longitude, orderRecord)
				if err == nil {
					distanceM = computedDistance
				}
				return err
			}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/order/service.go", "w") as f:
    f.write(content)

