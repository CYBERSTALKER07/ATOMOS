# V.O.I.D. Ecosystem Purpose & Protocol

## Primary Directive (F.R.I.D.A.Y.)
This is an advanced logistics ecosystem overseeing multi-role operations: Backend, Admin, Driver, Retailer, Payload, Factory, and Warehouse. 
The primary protocol is **End-to-End Coherence**: A role is a product, not an app. Any change requested MUST be synchronized across all connecting layers (Backend, Web, Mobile Native, Terminal, Kafka).

## Role Alignment
- **SUPPLIER (`role=ADMIN`)**: Operates the Admin Portal. There is no separate "platform admin" - the Admin Portal IS the Supplier Portal.
- **DRIVER (`role=DRIVER`)**: Route execution, geofenced actions, delivery verification. Scoped to a Home Node (Warehouse or Factory).
- **RETAILER (`role=RETAILER`)**: Order receipt, payment, demand feedback. Self-registered outside supplier scopes.
- **PAYLOAD (`role=PAYLOAD`)**: Loading, offloading, manifest confirmation via Terminal (Expo) / iOS / Android.
- **FACTORY/WAREHOUSE ADMIN**: Node-scoped administration for logistics generation and supply caching.

## The V.O.I.D. Entity Lifecycle
- Every business object rests on strong consistency and immutable tracing.
- Double-Entry Ledger: Every money movement writes two rows atomically (debit + credit) in the exact same Spanner transaction as the business state change.
- Strict Spanner / Role enforcement: Never trust request parameters for `supplier_id`, `factory_id`, etc., always derive from the JWT claims.