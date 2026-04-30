// Package pin provides global PIN generation, uniqueness enforcement, and
// lifecycle management for all authenticated entity types (Drivers,
// WarehouseStaff, FactoryStaff).
//
// Uniqueness is enforced via the GlobalPins Spanner table, which stores a
// deterministic SHA-256 hash of each plaintext PIN. bcrypt hashes (salted,
// non-deterministic) remain in the entity tables for authentication; the
// SHA-256 here is strictly for collision detection.
//
// All PIN writes MUST go through Registry.GenerateUnique inside a
// spanner.ReadWriteTransaction to guarantee atomicity with the entity row.
package pin
