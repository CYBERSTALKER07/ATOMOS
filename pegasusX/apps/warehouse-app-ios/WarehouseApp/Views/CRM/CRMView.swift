import SwiftUI

struct CRMView: View {
    @State private var vm = CRMViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.retailers.isEmpty {
                ContentUnavailableView(
                    "No Retailers",
                    systemImage: "storefront",
                    description: Text("mobile_warehouse.ui.no_retailer_relationships")
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.retailers) { retailer in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(retailer.name)
                                    .font(.headline)
                            }
                            Spacer()
                            VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                                Text(L10n.format("mobile_warehouse.ui.totalorders_orders", "\(retailer.totalOrders)"))
                                    .font(.caption)
                                Text(L10n.format("mobile_warehouse.ui.formatted_uzs", "\(retailer.totalRevenue.formatted())"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !retailer.lastOrderDate.isEmpty {
                                    Text(retailer.lastOrderDate)
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        .frame(minHeight: 44)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(retailer.name), \(retailer.totalOrders) orders, revenue \(retailer.totalRevenue.formatted()) UZS")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.retailers")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh retailers")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
