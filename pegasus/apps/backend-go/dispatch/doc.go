// Package dispatch consolidates the auto-dispatch, bin-packing, spatial
// clustering, and manifest-splitting algorithms used by Supplier, Warehouse,
// and Factory scopes. All pure-computation functions live here; HTTP handlers
// stay in their owning domain packages and delegate to dispatch.Service.
//
// Responsibilities:
//   - Spatial clustering (K-Means Lloyd's algorithm)
//   - Bin-packing (decreasing first-fit with H3 cell grouping)
//   - Vehicle selection (smallest-fit escalation)
//   - Manifest splitting (Rule of 25)
//   - Haversine distance, centroid, volumetric math
//   - Freeze-lock pre-check before dispatch execution
//
// Does NOT own: HTTP handlers, Spanner schema, Kafka topics, auth middleware.
package dispatch
