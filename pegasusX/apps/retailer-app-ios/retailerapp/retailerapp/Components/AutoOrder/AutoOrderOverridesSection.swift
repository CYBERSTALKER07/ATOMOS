import SwiftUI

struct OverrideItem: Identifiable {
    let id: String
    let label: String
    let enabled: Bool
    let hasHistory: Bool
    let level: OverrideLevel
}

enum OverrideLevel {
    case supplier, category, product, variant

    var subtitle: String {
        switch self {
        case .supplier: return "Supplier-level override"
        case .category: return "Category-level override"
        case .product: return "Product-level override"
        case .variant: return "Variant / SKU override"
        }
    }
}

struct AutoOrderOverridesSection: View {
    let title: String
    let icon: String
    let items: [OverrideItem]
    @Binding var localToggleStates: [String: Bool]
    let onToggle: (OverrideItem, Bool) -> Void

    var body: some View {
        LabCardWithHeader(title: title, icon: icon) {
            VStack(spacing: 0) {
                ForEach(items) { item in
                    OverrideToggleRow(
                        item: item,
                        externalValue: localToggleStates[item.id] ?? item.enabled,
                        onUserToggle: { newVal in
                            localToggleStates[item.id] = newVal
                            onToggle(item, newVal)
                        }
                    )

                    if item.id != items.last?.id {
                        Rectangle()
                            .fill(AppTheme.separator.opacity(0.2))
                            .frame(height: AppTheme.separatorHeight)
                    }
                }
            }
        }
    }
}

private struct OverrideToggleRow: View {
    let item: OverrideItem
    let externalValue: Bool
    let onUserToggle: (Bool) -> Void

    @State private var toggleValue = false

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            VStack(alignment: .leading, spacing: 2) {
                Text(item.label)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Text(item.level.subtitle)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            Toggle("", isOn: $toggleValue)
                .tint(AppTheme.accent)
                .labelsHidden()
                .scaleEffect(0.85)
                .accessibilityLabel("\(item.label), \(item.level.subtitle)")
        }
        .frame(minHeight: 44)
        .padding(.vertical, AppTheme.spacingSM)
        .padding(.horizontal, AppTheme.spacingXS)
        .onAppear {
            toggleValue = externalValue
        }
        .onChange(of: externalValue) { _, newValue in
            if toggleValue != newValue {
                toggleValue = newValue
            }
        }
        .onChange(of: toggleValue) { _, newValue in
            guard newValue != externalValue else { return }
            Haptics.selection()
            onUserToggle(newValue)
        }
    }
}
