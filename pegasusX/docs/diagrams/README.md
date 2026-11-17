# PegasusX Ecosystem Architecture & Lifecycle Diagrams

This directory contains standalone Mermaid (`.mmd`) diagram specifications documenting the complete order lifecycle, role user flows, and multi-surface architectures across the PegasusX multi-cell operating system.

## Index of Diagrams

| File | Diagram Type | Scope / Description |
| :--- | :--- | :--- |
| [`order_lifecycle.mmd`](order_lifecycle.mmd) | Sequence Diagram | **End-to-End Order Lifecycle**: 18 granular states from checkout, Cloud Spanner ACID locks, WMS wave planning, DVIR, doorstep delivery, Soliq fiscalization, blind cashier turn-in, to the settlement waterfall. |
| [`multi_agent_review_framework.mmd`](multi_agent_review_framework.mmd) | Flowchart | **25-Agent Multi-Council Review Framework**: 5 councils (Core Architecture, Statutory Compliance, Logistics & Yard, FinOps & Security, Client Surfaces). |
| [`control_tower_desktop_macos_architecture.mmd`](control_tower_desktop_macos_architecture.mmd) | Flowchart | **Desktop & macOS Native Architecture**: Multi-cell Spanner backplanes connected to Tauri 2 (Rust core) and macOS Native (Swift 6/SwiftUI, Swift Charts, MenuBarExtra). |
| [`user_flow_role_1_platform_owner.mmd`](user_flow_role_1_platform_owner.mmd) | Flowchart | **Role 1 User Flow: Platform Owner / Super Admin**: 5-phase flow: Root CLI/MFA provisioning, multi-cell EKG latency monitoring, dual-control maker-checker flag governance, Spanner outbox DLQ triage & replay, and FinOps take-rate ledger. |
| [`user_flow_role_2_supplier.mmd`](user_flow_role_2_supplier.mmd) | Flowchart | **Role 2 User Flow: Supplier / Brand Executive**: 5-phase flow: Tenant registration & 1C/SAP integration, catalog MXIK tax classification & tiered pricing, retailer credit profile & allocation, Foundry 5-stage blast-radius recall sandbox, and factoring payout disbursal. |
| [`user_flow_role_3_factory.mmd`](user_flow_role_3_factory.mmd) | Flowchart | **Role 3 User Flow: Factory / Production Manager**: 5-phase flow: Plant setup & industrial scanner pairing, BOM explosion & work order planning, automated GS1/SSCC pallet serialization, QA lab certification & quarantine locks, and loading bay transfer manifest. |
| [`user_flow_role_4_warehouse_manager.mmd`](user_flow_role_4_warehouse_manager.mmd) | Flowchart | **Role 4 User Flow: Warehouse Manager**: 5-phase flow: DC layout & Zebra RF DataWedge pairing, inbound ASN receiving & FEFO slotting, wave planning & snake-walk pick dispatch, BLE cold-chain sensor verification, and truck manifest bolt sealing. |
| [`user_flow_role_5_warehouse_cashier.mmd`](user_flow_role_5_warehouse_cashier.mmd) | Flowchart | **Role 5 User Flow: Warehouse Cashier**: 5-phase flow: Cash desk terminal & opening float setup, driver tamper-evident bag intake, zero-trust blind denomination counting (200k..1k UZS), atomic Spanner variance resolution & dispute escalation, and CIT armored drop-safe handover. |
| [`user_flow_role_6_yard_marshal.mmd`](user_flow_role_6_yard_marshal.mmd) | Flowchart | **Role 6 User Flow: Yard Marshal & Dock Master**: 5-phase flow: Ruggedized tablet & ALPR camera calibration, optical license plate clearance, 8-point pre-trip DVIR safety gating, high-security bolt seal inspection & weighbridge gross axle check, and dock turnaround (< 10 min) dwell time tracking. |
| [`user_flow_role_7_driver_carrier.mmd`](user_flow_role_7_driver_carrier.mmd) | Flowchart | **Role 7 User Flow: Carrier & Fleet Driver**: 5-phase flow: CDL onboarding & peripheral pairing, 8-point photographic pre-trip DVIR inspection, H3 turn-by-turn vector navigation, 100m geofence doorstep unlock with velocity anti-spoofing guard, and depot blind bag cash return. |
| [`user_flow_role_8_retailer.mmd`](user_flow_role_8_retailer.mmd) | Flowchart | **Role 8 User Flow: Retailer & Merchant Partner**: 5-phase flow: Soliq TIN registration & closest warehouse DC discovery, wholesale catalog browse with dynamic volume discounts, unified ParentOrders checkout saga with uncapped multi-tender checkout (no credit or payment limits), live delivery radar & doorstep inspection, and dynamic QR handshake with store POS restocking. |
| [`user_flow_role_9_sre_finops.mmd`](user_flow_role_9_sre_finops.mmd) | Flowchart | **Role 9 User Flow: Enterprise SRE & FinOps Lead**: 5-phase flow: Workstation enclave & multi-cloud IAM setup, native macOS MenuBarExtra pulse monitoring, Kafka consumer partition lag & worker pool scaling, Spanner DLQ diagnostic export & atomic replay, and tenant-attributed cloud cost waterfall. |

---

## Viewing & Rendering

These `.mmd` files use standard Mermaid syntax and can be rendered using:
- **Mermaid CLI (`mmdc`)**: `npx @mermaid-js/mermaid-cli -i order_lifecycle.mmd -o order_lifecycle.svg`
- **GitHub Markdown**: Fenced code blocks with language `mermaid`
- **VS Code / Cursor / IDEs**: Using Mermaid Preview extensions or Markdown preview
