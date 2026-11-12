import SwiftUI
import UIKit

struct ReportsProView: View {
    @State private var vm = ReportsProViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            if let loadError = vm.loadError {
                Section {
                    Text(loadError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.destructive)
                    Button("common.action.retry") {
                        Haptics.light()
                        Task { await vm.load() }
                    }
                    .pegasusMinimumHitTarget()
                    .accessibilityLabel("Retry loading reports")
                }
            } else if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }

            if vm.summaryReady && vm.loadError == nil {
                Section("Last 7 days") {
                    Text("Sales \(vm.formatMinor(vm.salesMinor))")
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Sales \(vm.formatMinor(vm.salesMinor))")
                    Text(L10n.format("mobile_retailer.ui.sale_count_salecount_2", "\(vm.saleCount)"))
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Sale count \(vm.saleCount)")
                    Text(L10n.format("mobile_retailer.ui.on_hand_skus_onhand_2", "\(vm.onHand)"))
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("On hand SKUs \(vm.onHand)")
                    Text(L10n.format("mobile_retailer.ui.low_stock_bins_lowstock_2", "\(vm.lowStock)"))
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Low stock bins \(vm.lowStock)")
                    Text(L10n.format("mobile_retailer.ui.top_sku_topline_2", "\(vm.topLine)"))
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Top SKU \(vm.topLine)")
                }
            }

            Section {
                Button {
                    Task { await vm.exportCSV() }
                } label: {
                    HStack {
                        Text(vm.exporting ? "Exporting…" : "Export sales CSV")
                        Spacer()
                        if vm.exporting {
                            ProgressView()
                        }
                    }
                    .pegasusMinimumHitTarget()
                }
                .disabled(vm.exporting)
                .accessibilityLabel("Export sales CSV")
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.reports_pro")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.load() }
        .refreshable { await vm.load() }
        .sheet(isPresented: $vm.showShare) {
            if let exportURL = vm.exportURL {
                ActivityView(activityItems: [exportURL])
            }
        }
    }
}

/// UIKit share sheet bridge for CSV files.
private struct ActivityView: UIViewControllerRepresentable {
    let activityItems: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: activityItems, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
