# Apple Design Context — PegasusX Native Ecosystem

## Product
- **Name**: PegasusX Native Control Tower & Operations Suite
- **Description**: Mission-critical B2B commerce and logistics operating system for multi-tenant retail, wholesale, warehouse depots, and carrier fleets across Uzbekistan (Central Asia).
- **Category**: Enterprise Logistics, Telemetry & FinOps
- **Stage**: Production / Active Hardening

## Platforms
| Platform | Supported | Min OS | Notes |
|----------|-----------|--------|-------|
| macOS    | Yes       | 14.0+  | Native Control Tower (`apps/control-tower-macos`): NavigationSplitView, Swift Charts, MenuBarExtra, ⌘ keyboard shortcuts |
| iOS      | Yes       | 17.0+  | Native Warehouse (`apps/warehouse-app-ios`), Driver (`apps/driver-app-ios`), Supplier (`apps/supplier-app-ios`), Payload (`apps/payload-app-ios`) |
| iPadOS   | Yes       | 17.0+  | Adaptive three-column and multi-panel tactical workbench |
| watchOS  | No        | —      | Secondary consideration; glanceable dispatch notifications via APNs |
| visionOS | No        | —      | Not in active scope |
| tvOS     | No        | —      | Dedicated dashboard mode handled via macOS Control Tower or web displays |

## Technology
- **UI Framework**: SwiftUI (100% native declarative UI) with selective AppKit / UIKit bridging for low-level system integrations (NSStatusItem, UIImpactFeedbackGenerator).
- **Architecture**:
  - macOS: Unified window (`NavigationSplitView`) + Top `MenuBarExtra` system status accessory.
  - iOS: NavigationStack, bottom tab bars, modal sheets for non-blocking secondary operations.
- **Apple Technologies**:
  - **Swift Charts**: High-performance multi-series time-series latency plots and tender settlement waterfall bars.
  - **Core Haptics / UIFeedbackGenerator**: Tactile feedback on physical counting, checklist verification, and quarantine gating.
  - **Live Activities / ActivityKit**: Active route dispatch tracking and live order lifecycle progression.
  - **VoiceOver / Accessibility**: Full accessibility labels, values, traits, and monospaced tabular figures.

## Design System
- **Base**: Custom Pegasus Monochrome Tactical Design System (`TacticalTheme` / `PegasusMonochromeTheme`) aligned with Apple Human Interface Guidelines.
- **Brand Colors**:
  - Deep Canvas: `0x09090B` (Dark) / `0xF8FAFC` (Light)
  - Surface Cards: `.ultraThinMaterial` / `.secondarySystemGroupedBackground`
  - Cobalt Accent: `0x3B82F6` (Primary action)
  - Success Green: `0x10B981` (Balanced / Pass / Healthy)
  - Warning Orange: `0xF59E0B` (Recount required / In-transit)
  - Danger Red: `0xEF4444` (Defect / Red-tag quarantine / DLQ poison)
- **Typography**: Apple System Fonts (`SF Pro`, `SF Mono` for all currency amounts, Tiyin arithmetic, and telemetry latencies).
- **Dark Mode**: Fully supported with semantic dynamic colors (`Color(uiColor: .label)`, `Color(NSColor.controlBackgroundColor)`).
- **Dynamic Type**: Supported across all mobile views with scalable text styles and layout auto-wrapping.

## Accessibility
- **Target Level**: WCAG 2.1 / 2.2 AA and Apple Baseline Accessibility.
- **Key Considerations**:
  - Minimum touch targets of $44 \times 44\,\text{pt}$ on iOS and $48 \times 48\,\text{dp}$ on Android.
  - High-contrast visual indicators accompanying all color cues (icons + text capsules, never color alone).
  - VoiceOver semantic labels, values, and traits for banknote counters and safety inspection rows.
  - Zero layout shift during live count updates via `.monospacedDigit()`.
  - Respect for `accessibilityReduceMotion` across all animations.

## Users & Work Environments
- **Primary Personas**:
  1. *Depot Cashier & Vault Marshal*: Fast physical counting of large stacks of UZS banknotes in dimly lit depot cash cages; needs immediate tactile feedback and high-contrast legible figures.
  2. *Fleet Driver & Carrier*: In-cab truck walkaround vehicle inspections wearing gloves; needs oversized touch targets and rapid pass/defect toggles.
  3. *Enterprise SRE & FinOps Operator*: High-density multi-monitor command center; needs keyboard shortcuts, live background status bar glanceability, and zero-latency telemetry.
- **Key Challenges**:
  - Preventing cashier confirmation bias through strict dual-blind masking.
  - Guaranteeing safety compliance before dispatch without slowing down warehouse turnaround.
  - Displaying complex distributed Spanner latencies and Kafka queue depths without visual clutter.
