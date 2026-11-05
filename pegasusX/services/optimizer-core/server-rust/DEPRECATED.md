# Deprecation Notice: Rust Solver Service (server-rust)

**Status**: DEPRECATED  
**Canonical Production Solver**: Python OR-Tools Solver in [`services/optimizer-core/server/`](../server)

### Rationale
1. **Zero Production Traffic**: The backend Go service (`apps/backend-go/dispatch/optimizerclient/client.go`) routes 100% of VRP optimization traffic directly to the Python OR-Tools engine on port 8082 via `POST /v1/optimizer/solve`.
2. **Solver Parity**: Google OR-Tools CP-SAT and Routing engines are implemented in native C++ under the hood with bindings in Python, providing full support for vehicle capacity dimensions, dynamic time windows, dropped visit penalties, and Guided Local Search (GLS).
3. **No Active Deployment**: This service is excluded from Docker Compose and Kubernetes production topologies.
