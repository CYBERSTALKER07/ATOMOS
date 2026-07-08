cat << 'MD' > /Users/shakhzod/.gemini/antigravity/brain/795db57e-0878-4d53-bedb-4c35a9c85e6c/task.md
# Tasks

- `[x]` 1. Smart Dispatch: Balanced Routing Optimization
  - `[x]` Update `BinPack` to use balanced distribution algorithm
  - `[x]` Implement fallback logic for retailer-level consolidated order fallback
- `[x]` 2. Payloader & Warehouse UI: Dynamic Complete/Partial Reassign
  - `[x]` Implement dynamic Complete Reassignment logic (no order split)
  - `[x]` Implement dynamic Partial Reassignment (Split) tracking Q loaded onto each truck
- `[x]` 3. Cross-Truck Synchronization: On-the-Way & Payment Coordination
  - `[x]` Implement WebSocket trigger for "On the way" event
  - `[x]` Update Payment sync logic to update shared Order state
  - `[x]` Add "Start Transit" UI for split orders in Driver app
- `[/]` 4. Phased Execution for Cross-Role App Feature Parity
  - `[/]` Phase A: Supplier & Retailer Parity
    - `[ ]` Retailer: Implement full Procurement parity (iOS & Android)
    - `[ ]` Retailer: Implement Setup Wizard in native apps (iOS & Android)
    - `[ ]` Supplier: Implement full "empathy adoption" depth
  - `[ ]` Phase B: Warehouse & Factory Parity
    - `[ ]` Factory iOS: Convert Analytics from bottom sheet to full dedicated tab
    - `[ ]` Factory iOS: Convert Exceptions from bottom sheet to full dedicated tab
    - `[ ]` Warehouse: Build out "supply forecast create form"
    - `[ ]` Factory: Implement full Treasury depth
  - `[ ]` Phase C: Driver & Payloader Parity
    - `[ ]` Driver: Deferred negotiation states (if applicable)
- `[ ]` 5. Verification
  - `[ ]` Verify compilation for all updated apps
MD
