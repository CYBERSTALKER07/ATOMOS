# Unified Design System Master File (V.O.I.D & Pegasus Master Standard)

> [!IMPORTANT]
> **BINDING AGENT DIRECTIVE**: All pages, components, and layouts MUST strictly adhere to:
> - Master System Rules: [.agents/rules/ui-design-system.md](file:///Users/shakhzod/Desktop/V.O.I.D/.agents/rules/ui-design-system.md)
> - Master Design Guide: [DESIGN.md](file:///Users/shakhzod/Desktop/V.O.I.D/DESIGN.md)

---

## 1. Visual Language & Token Mappings

| Role / Token | Dark Value (Tactical Canvas) | Light Value (Operational Canvas) | CSS Variable |
|:---|:---|:---|:---|
| Canvas | `#09090B` | `#F8FAFC` | `--desk-canvas` |
| Surface (Card) | `#121216` | `#FFFFFF` | `--desk-surface` |
| Surface Subtle | `#181820` | `#F1F5F9` | `--desk-surface-subtle` |
| Surface Sunken | `#0D0D11` | `#E2E8F0` | `--desk-surface-sunken` |
| Border | `#22222C` | `#E2E8F0` | `--desk-border` |
| Border Strong | `#333342` | `#CBD5E1` | `--desk-border-strong` |
| Text Primary | `#F8FAFC` | `#0F172A` | `--desk-text-primary` |
| Text Secondary | `#94A3B8` | `#475569` | `--desk-text-secondary` |
| Text Tertiary | `#64748B` | `#94A3B8` | `--desk-text-tertiary` |
| Accent Cobalt | `#3B82F6` | `#2563EB` | `--desk-accent` |
| Accent Orange | `#FF7A1A` | `#F97316` | `--desk-accent-orange` |
| Accent Lime | `#E2FD52` | `#65A30D` | `--desk-accent-lime` |
| Accent Purple | `#8B5CF6` | `#7C3AED` | `--desk-accent-purple` |
| Accent Cyan | `#06B6D4` | `#0284C7` | `--desk-accent-cyan` |
| Success Mint | `#10B981` | `#059669` | `--desk-success` |
| Warning Amber | `#F59E0B` | `#D97706` | `--desk-warning` |
| Danger Crimson| `#EF4444` | `#DC2626` | `--desk-danger` |

## 2. Mandatory Component Conventions

1. **Tabular Monospace**: All numbers, quantities, money, percentages, ETAs, plates, and timestamps MUST use `font-mono tabular-nums`.
2. **Hairline Precision**: 1px borders only (`border border-[var(--desk-border)]`). No heavy, blurry dark shadows.
3. **Status Badges**: Always use a tactical pill with status dot (`●`) and monospace text.
4. **Layout**: 3-Column Control Tower layout (Nav Rail → Operations Stage / Feed → Inspector Drawer) & floating command bar.
