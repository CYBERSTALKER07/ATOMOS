import SwiftUI

struct OrderChip: View {
    let order: LiveOrder
    let selected: Bool
    let sealed: Bool
    let dispatchCode: String?
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 12) {
                if sealed {
                    Image(systemName: "checkmark.seal.fill")
                        .font(.system(.title3))
                        .foregroundStyle(TermTheme.live)
                } else {
                    Image(systemName: "circle.dotted")
                        .font(.system(.title3))
                        .foregroundStyle(selected ? TermTheme.accent : TermTheme.tertiary)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(order.orderId.suffix(6).uppercased())"))
                        .font(.system(.subheadline, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.accent)

                    Text(L10n.format("mobile_payload.ui.count_units", "\((order.items ?? []).count)"))
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .foregroundStyle(TermTheme.secondary)
                }

                Spacer()

                if sealed, let code = dispatchCode, !code.isEmpty {
                    Text(code)
                        .font(.system(.callout, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.live)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(TermTheme.live.opacity(0.12), in: Capsule())
                }
            }
            .padding(14)
            .frame(minWidth: 44, minHeight: 48)
            .background {
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .fill(background)
                    .overlay {
                        if selected {
                            RoundedRectangle(cornerRadius: 16, style: .continuous)
                                .stroke(TermTheme.accent.opacity(0.2), lineWidth: 2)
                        } else {
                            RoundedRectangle(cornerRadius: 16, style: .continuous)
                                .stroke(TermTheme.separator.opacity(0.08), lineWidth: 1)
                        }
                    }
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Order \(order.orderId.suffix(6)), \((order.items ?? []).count) units, \(sealed ? "sealed with code \(dispatchCode ?? "")" : "unsealed")")
        .accessibilityAddTraits(selected ? [.isSelected] : [])
        .buttonStyle(.tactical)
    }

    private var background: Color {
        if sealed { return TermTheme.live.opacity(0.04) }
        if selected { return TermTheme.accent.opacity(0.08) }
        return TermTheme.card
    }
}

struct ItemRow: View {
    let checked: Bool
    let enabled: Bool
    let label: String
    let quantity: Int
    let onToggle: () -> Void

    var body: some View {
        Button(action: { if enabled { onToggle() } }) {
            HStack(spacing: 14) {
                ZStack {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(checked ? TermTheme.accent : TermTheme.accent.opacity(0.06))
                        .frame(width: 32, height: 32)

                    if checked {
                        Image(systemName: "checkmark")
                            .font(.system(.subheadline, weight: .black))
                            .foregroundStyle(TermTheme.card)
                    }
                }

                Text(label.uppercased())
                    .font(.system(.footnote, design: .monospaced, weight: .bold))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .foregroundStyle(checked ? TermTheme.accent : TermTheme.secondary)

                Text(L10n.format("mobile_payload.ui.qty_quantity_2", "\(quantity)"))
                    .font(.system(.footnote, design: .monospaced, weight: .black))
                    .foregroundStyle(TermTheme.accent)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(TermTheme.accent.opacity(0.06), in: RoundedRectangle(cornerRadius: 6, style: .continuous))
            }
            .frame(minWidth: 44, minHeight: 44)
            .padding(.vertical, 8)
            .padding(.horizontal, 12)
            .background {
                if checked {
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .fill(TermTheme.accent.opacity(0.03))
                }
            }
            .contentShape(Rectangle())
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label), quantity \(quantity), \(checked ? "checked" : "not checked")")
        .accessibilityAddTraits(checked ? [.isSelected] : [])
        .buttonStyle(.tactical)
        .opacity(enabled ? 1 : 0.5)
        .disabled(!enabled)
    }
}
