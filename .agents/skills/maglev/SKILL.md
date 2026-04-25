---
name: maglev
description: "Use when: implementing, auditing, or explaining Maglev-style software load balancing, consistent hashing, lookup table generation, VIP routing, ECMP, BGP announcement, DSR/GRE forwarding, connection tracking, health checks, failover, session affinity, Google Cloud load balancer selection, backend services, forwarding rules, target proxies, Cloud Armor, observability, IAM roles, or hyperscale traffic distribution in V.O.I.D."
---

# Maglev Load Balancing Skill

## Purpose

Use this skill to design, implement, review, or troubleshoot Maglev-style load balancing and adjacent Google Cloud load balancing flows. The goal is to preserve three invariants:

1. High throughput: packet or request forwarding must stay on a low-overhead path.
2. Stability: backend changes should remap the smallest practical number of flows.
3. Operability: health, failover, observability, security, and rollout behavior must be explicit.

The complete imported source material from the original draft is preserved at `references/source-material.md`. Load that file only when a deep reference pass is needed; it is intentionally not the main skill body because it is very large.

## When This Applies

Use this skill for:

- Maglev consistent-hashing table design or debugging.
- Load balancer implementations that use VIPs, backend pools, ECMP, BGP, GRE encapsulation, or Direct Server Return.
- Backend pool health checks, failover, drain, and endpoint churn behavior.
- Session affinity, connection tracking, and per-flow stability.
- Google Cloud load balancer selection and architecture reviews.
- Security or operations reviews for load balancing: Cloud Armor, IAM, logging, monitoring, autoscaling, and topology choices.
- V.O.I.D. hyperscale request distribution, stateless pod routing, WebSocket/Kafka traffic paths, and edge routing decisions.

Do not use this skill for unrelated CRUD routing, ordinary reverse-proxy configuration, or UI-only tasks.

## Source Map

The bundled source material contains these major areas:

- Maglev paper: software load balancer motivation, packet flow, forwarder architecture, connection tracking, consistent hashing, operational experience, evaluation, related work, and conclusion.
- Google Cloud Load Balancing overview: product families, global/regional choices, internal/external choices, proxy versus passthrough behavior, backend services, health checks, forwarding rules, URL maps, certificates, firewall rules, monitoring, logging, security, IAM, and integrations.
- Load balancer selection guidance: traffic type, protocols, network topology, failover, backend type, session affinity, autoscaling, and autohealing.

## Maglev Architecture Checklist

When designing or auditing a Maglev-like path, verify each layer:

1. VIP ownership: each virtual IP maps to an explicit service or backend pool.
2. Router distribution: upstream routers spread traffic across load balancer machines through ECMP or equivalent hashing.
3. Load balancer health: each machine announces or withdraws VIPs based on forwarder health.
4. Backend health: endpoints are eligible only after passing the configured health checks.
5. Flow selection: each packet or request hashes stable flow identity, usually the 5-tuple for L4 traffic.
6. Connection persistence: existing flows prefer their connection-tracking entry while the selected backend remains healthy.
7. Consistent hashing fallback: new or invalidated flows use the Maglev lookup table.
8. Encapsulation or forwarding: selected traffic is forwarded using the platform mechanism, such as GRE, proxying, or passthrough.
9. Return path: DSR or proxy response behavior is understood and documented.
10. Observability: logs, metrics, health state, backend selection, drops, failovers, and table versions are inspectable.

## Consistent Hashing Table Rules

Maglev lookup tables must be deterministic and shared by every load balancer in the same shard.

Use this shape:

1. Choose a lookup table size `M` that is prime and much larger than the backend count.
2. For every backend, compute two hash-derived values:
   - `offset = h1(backend) % M`
   - `skip = h2(backend) % (M - 1) + 1`
3. Generate each backend permutation with:
   - `permutation[j] = (offset + j * skip) % M`
4. Fill the table in backend round-robin order, placing each backend into its next open permutation slot.
5. Look up a flow by hashing the flow key and indexing `lookup[hash % M]`.

Review requirements:

- Table generation must be deterministic across processes and languages.
- Backend ordering must be stable; accidental reorder can remap many flows.
- Backend add/remove should only remap the expected subset of flows.
- Table publication must be atomic; readers must never see a half-filled table.
- Existing flow entries must be validated against current backend health.

## Connection Tracking Rules

Connection tracking complements consistent hashing; it does not replace it.

- Track recent flow key to backend decisions per worker/thread when possible to avoid lock contention.
- Reuse a tracked backend only while that backend is still healthy and eligible for the VIP.
- Expire idle entries to cap memory.
- On backend failure, bypass stale entries and reselect through the Maglev table.
- Avoid shared mutable connection maps without explicit synchronization or per-shard ownership.

## Health And Failover Rules

Health affects two separate layers:

- Load balancer machine health controls VIP announcement or withdrawal.
- Backend endpoint health controls selection eligibility.

Audit for:

- Fast withdrawal when a forwarder is unhealthy.
- Deduplicated health checks when the same endpoint appears in multiple pools.
- Drain behavior before removing a backend when policy allows.
- Clear behavior for zero healthy backend cases: drop, fail closed, return 503, or route to failover pool.
- Consistent config rollout across machines, with tolerance for short propagation windows.

## Google Cloud Load Balancing Decision Guide

Pick the load balancer by traffic and control-plane needs:

- External Application Load Balancer: internet-facing HTTP(S), URL maps, TLS termination, Cloud CDN, Cloud Armor, advanced routing.
- Internal Application Load Balancer: private HTTP(S) services inside a VPC.
- Proxy Network Load Balancer: TCP/SSL proxy behavior with connection termination at the proxy layer.
- Passthrough Network Load Balancer: L3/L4 forwarding where backends see original client packets and proxy semantics are not desired.
- Regional load balancing: regional workloads, local compliance, or zonal backend affinity.
- Global load balancing: anycast/global frontend, cross-region failover, or worldwide latency steering.

Always document:

- External versus internal.
- Global versus regional.
- Proxy versus passthrough.
- Client protocol and backend protocol.
- Backend type: instance group, network endpoint group, serverless NEG, bucket, or service mesh integration.
- Health check type and firewall requirements.
- Session affinity policy.
- Logging, metrics, alerting, and SLOs.
- Security controls: Cloud Armor, TLS policy, certificates, IAM roles, firewall rules, and private connectivity.

## V.O.I.D. Hyperscale Fit

For V.O.I.D. work, tie Maglev decisions back to the platform invariants:

- Stateless pods: routing must not require sticky local state unless backed by Redis or another distributed primitive.
- WebSockets: connection placement can be hash-stable, but cross-pod delivery must still use the configured hub relay.
- Kafka: partition keys preserve per-aggregate order; load balancing must not break consumer group semantics.
- Redis rate limits: edge routing should preserve actor identity for per-actor token buckets.
- Priority guard: overload behavior must shed low-priority work quickly instead of queueing indefinitely.
- Circuit breakers: upstream failure should fast-fail and avoid cascading dependency collapse.
- Observability: every request path needs trace IDs, load balancer logs, backend health state, and saturation metrics.

## Implementation Review Checklist

Before declaring Maglev or load-balancing work complete, verify:

- Config updates are atomic and versioned.
- Lookup table generation is deterministic and covered by tests.
- Backend add/remove tests measure remap impact.
- Health transitions do not leave stale eligible endpoints.
- Connection tracking handles backend death and idle expiry.
- The routing path is bounded: no unbounded goroutines, queues, maps, or retries.
- Request or packet distribution is observable by VIP, backend pool, backend, zone/region, and error class.
- Failover behavior is tested for single-backend, multi-backend, and all-backends-down cases.
- Security controls are explicit: IAM, firewall rules, TLS/certs if applicable, and DDoS/WAF controls for external surfaces.
- Rollback is possible by reverting config or table version without process restart where practical.

## Testing Guidance

Use focused tests for:

- Deterministic table output from fixed backend inputs.
- Even distribution across healthy backends.
- Minimal disruption when adding or removing one backend.
- Stable selection for repeated flow keys.
- Stale connection-tracking entries after backend health changes.
- Zero healthy backend behavior.
- Atomic config swap: readers see old or new config, never partial state.
- Concurrent readers during table rebuilds.

For V.O.I.D. backend changes, also run the relevant build or test command from the canonical monorepo path and check connected consumers before closing the task.

## Deep Reference Procedure

When the compact checklist is not enough:

1. Open `references/source-material.md`.
2. Locate the relevant source section with search rather than loading the whole file.
3. Extract the operational rule into the implementation or review.
4. Keep the skill body compact; add only durable, reusable rules here.
