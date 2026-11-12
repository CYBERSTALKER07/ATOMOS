import SwiftUI

struct SupplySummaryCard: View {
    let total: Int
    let visible: Int
    let runtimeStatus: String
    let runtimeTone: FactoryRuntimeTone

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("mobile_factory.ui.warehouse_demand_queue")
                .font(.title2.bold())
            Text(L10n.format("mobile_factory.ui.visible_requests_in_view_total_total_across_the_factory_queue_3", "\(visible)", "\(total)"))
                .font(.body)
                .foregroundStyle(.secondary)
            FactoryRuntimeBanner(tone: runtimeTone, message: runtimeStatus)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Warehouse demand queue, \(visible) visible of \(total) total. \(runtimeStatus)")
    }
}

struct SupplyFilterAndModeBar: View {
    @Binding var selectedFilter: String
    @Binding var viewMode: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: LabTheme.spacingSM) {
                    ForEach(supplyRequestFilters, id: \.self) { filter in
                        Button {
                            PegasusHaptics.selection()
                            selectedFilter = filter
                        } label: {
                            Text(filter.replacingOccurrences(of: "_", with: " "))
                                .font(.footnote.bold())
                                .padding(.horizontal, 12)
                                .padding(.vertical, 8)
                                .frame(minHeight: 44)
                                .background(selectedFilter == filter ? LabTheme.label : Color.clear, in: Capsule())
                                .foregroundStyle(selectedFilter == filter ? Color(.systemBackground) : LabTheme.label)
                                .overlay(Capsule().stroke(.quaternary))
                                .contentShape(Capsule())
                        }
                        .buttonStyle(PressableButtonStyle())
                        .accessibilityLabel("Filter \(filter.replacingOccurrences(of: "_", with: " "))")
                        .accessibilityAddTraits(selectedFilter == filter ? .isSelected : [])
                    }
                }
            }

            HStack(spacing: LabTheme.spacingSM) {
                modeButton("TABLE", label: "Table")
                modeButton("BOARD", label: "Board")
            }
        }
    }

    private func modeButton(_ mode: String, label: String) -> some View {
        Button {
            PegasusHaptics.selection()
            viewMode = mode
        } label: {
            Text(label)
                .font(.footnote.bold())
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .frame(minHeight: 44)
                .background(viewMode == mode ? LabTheme.label : Color.clear, in: Capsule())
                .foregroundStyle(viewMode == mode ? Color(.systemBackground) : LabTheme.label)
                .overlay(Capsule().stroke(.quaternary))
                .contentShape(Capsule())
        }
        .buttonStyle(PressableButtonStyle())
        .accessibilityLabel("View mode \(label)")
        .accessibilityAddTraits(viewMode == mode ? .isSelected : [])
    }
}
