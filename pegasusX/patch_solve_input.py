import re

with open('apps/backend-go/dispatch/optimizerclient/client.go', 'r') as f:
    content = f.read()

old_input = """type SolveInput struct {
	TraceID       string
	SupplierID    string
	HomeNodeID    string
	DepotLat      float64
	DepotLng      float64
	DepartureTime time.Time
	Orders        []dispatch.GeoOrder
	Fleet         []dispatch.AvailableDriver
}"""

new_input = """type SolveInput struct {
	TraceID       string
	SupplierID    string
	HomeNodeID    string
	DepotLat      float64
	DepotLng      float64
	DepartureTime time.Time
	Orders        []dispatch.GeoOrder
	Fleet         []dispatch.AvailableDriver
	TetrisBuffer  float64
}"""

if old_input in content:
    content = content.replace(old_input, new_input)
    print("Patched SolveInput.")
else:
    print("Failed to patch SolveInput.")

old_req = """		Tunables: &contract.Tunables{
			TimeLimitMs: DefaultSolverTimeLimitMs,
		},"""

new_req = """		Tunables: &contract.Tunables{
			TimeLimitMs:  DefaultSolverTimeLimitMs,
			TetrisBuffer: in.TetrisBuffer,
		},"""

if old_req in content:
    content = content.replace(old_req, new_req)
    print("Patched SolveRequest.")
else:
    print("Failed to patch SolveRequest.")

with open('apps/backend-go/dispatch/optimizerclient/client.go', 'w') as f:
    f.write(content)
