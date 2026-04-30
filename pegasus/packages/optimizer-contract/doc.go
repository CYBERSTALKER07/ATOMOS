// Package optimizercontract is the wire contract between the V.O.I.D. backend
// (client, in apps/backend-go) and the ai-worker dispatch optimiser (server,
// in apps/ai-worker). It is intentionally dependency-free — only the Go std
// library — so both sides can import it without dragging Spanner, Kafka, or
// any HTTP framework into the other module's build graph.
//
// Wire format: HTTP POST application/json. The optimiser server lives at
//
//	POST /v1/optimizer/solve
//
// guarded by a shared-secret header `X-Internal-Api-Key`. The backend client
// enforces a hard 2.5 s timeout; on timeout, network failure, or 5xx response
// it falls back to the Phase 1 K-Means + binpack pipeline.
//
// Versioning: the contract is locked to V (= "v1"). Wire-breaking changes
// bump V and live behind a new path (/v1/optimizer/solve → /v2/...). Field
// additions are backward-compatible if all writers populate the new field.
package optimizercontract
