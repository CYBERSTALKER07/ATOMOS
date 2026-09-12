import re

with open('apps/backend-go/dispatch/plan/optimize.go', 'r') as f:
    content = f.read()

old_block = """		in := optimizerclient.SolveInput{
			TraceID:    job.TraceID,
			SupplierID: job.SupplierID,
			HomeNodeID: job.HomeNodeID,
			DepotLat:   job.DepotLat,
			DepotLng:   job.DepotLng,
			Orders:     geoOrdersFromDispatchable(job.Orders),
			Fleet:      job.Fleet,
		}"""

new_block = """		in := optimizerclient.SolveInput{
			TraceID:      job.TraceID,
			SupplierID:   job.SupplierID,
			HomeNodeID:   job.HomeNodeID,
			DepotLat:     job.DepotLat,
			DepotLng:     job.DepotLng,
			Orders:       geoOrdersFromDispatchable(job.Orders),
			Fleet:        job.Fleet,
			TetrisBuffer: maxAcceptableUtilFraction,
		}"""

if old_block in content:
    content = content.replace(old_block, new_block)
    print("Patched optimize.go successfully.")
else:
    print("Failed to patch optimize.go.")

with open('apps/backend-go/dispatch/plan/optimize.go', 'w') as f:
    f.write(content)
