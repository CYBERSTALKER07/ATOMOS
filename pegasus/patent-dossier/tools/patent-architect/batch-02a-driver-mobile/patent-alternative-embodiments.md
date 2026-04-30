# Batch 02A - Driver Mobile Alternative Embodiments

## Embodiment Matrix

| Feature Family | Base Embodiment | Alternative A | Alternative B | Alternative C | Data Integrity Guard |
|---|---|---|---|---|---|
| Mission Routing | Optimized stop sequence rendered on device with optional manual next-stop override | Strict optimizer-only sequence with no override affordance | Hybrid mode where override allowed only after lock acquisition | Dispatch-center delegated override where driver sends request token | Route writes require freeze lock and version gate checks |
| Delivery Proof | QR scan + quantity confirmation + state commit | NFC token handshake at retailer dock | Bluetooth proximity + signed timestamp envelope | Camera OCR with hash challenge fallback | Completion commit remains geofence-gated and idempotent |
| Settlement Closure | Payment waiting state then success transition | Cash-only settlement with cashier OTP confirmation | Gateway-first settlement with delayed proof upload | Supplier pre-authorized settlement bypass for outage windows | Double-entry ledger pair and outbox emission in same transaction |
| Offline Resilience | Offline verifier mode with local encrypted queue | Local-only pending list with delayed sync timer | SMS fallback handshake for low-connectivity zones | Manual hotline escalation with supervisor token | Replay prevention by idempotency key and request hash |
| Exception Handling | Shop-closed waiting with escalation path | Auto-reschedule to next slot within policy bounds | Instant warehouse return authorization flow | Split completion with partial delivery closure | Exception updates emit immutable audit events |

## Continuity Notes

- All embodiments preserve the canonical driver order lifecycle and audit trace propagation.
- Alternatives vary interaction modality only; transactional safety and event lineage remain unchanged.
- Any embodiment that changes client payload shape remains additive to preserve backward compatibility.
