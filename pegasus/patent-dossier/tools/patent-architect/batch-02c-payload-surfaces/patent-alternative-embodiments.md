# Batch 02C - Payload Surfaces Alternative Embodiments

## Embodiment Matrix

| Feature Family | Base Embodiment | Alternative A | Alternative B | Alternative C | Data Integrity Guard |
|---|---|---|---|---|---|
| Manifest Composition | Manual order selection with guided checklist and scan verification | Barcode-first auto-queue by scanned bins | Voice-assisted line confirmation | Dock-station linked prefilled manifest with manual prune | Manifest state transitions validated by finite-state machine |
| Seal Confirmation | Post-seal countdown then dispatch success | Dual-operator confirmation with supervisor PIN | Hardware token seal authorization | Deferred seal with cryptographic timestamp and grace window | Seal action idempotent with request-hash replay protection |
| Truck Assignment | Worker picks truck from availability list | Recommendation-sorted truck candidates | Policy-locked truck assignment from dispatch center | Zone-based truck pools with fallback selection | Assignment remains scoped by home node and role claims |
| Offline Continuity | Local queue plus delayed sync | QR export packet for manual bridge | SMS fallback status marker | Paper fallback with later reconciliation entry | Replay-safe idempotent ingest on reconnect |
| Success Handoff | Success screen with dispatch code and restart option | Auto-print manifest code at dock kiosk | Dispatch code QR for driver pickup | Silent success with push confirmation to supervisor dashboard | Outbox guarantees downstream fanout without ghost states |

## Continuity Notes

- Terminal and native payload embodiments retain the same manifest lifecycle checkpoints.
- Alternatives primarily vary operator interaction modality and hardware integration.
- Each variant preserves deterministic audit traces and at-least-once-safe event delivery.
