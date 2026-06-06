// Package dispatch owns the shared dispatch read model and preview helpers
// used by supplier and warehouse operational surfaces.
//
// Receiving windows are resolved live from Retailers at fetch time so profile
// edits propagate to dispatch without rewriting historical order rows.
package dispatch
