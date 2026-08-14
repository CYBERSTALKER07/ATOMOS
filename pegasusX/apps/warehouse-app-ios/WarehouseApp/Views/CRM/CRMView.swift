import SwiftUI

struct CRMView: View {
    @State private var retailers: [Retailer] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { load() }
                    }
                } else if retailers.isEmpty {
                    ContentUnavailableView("No Retailers", systemImage: "storefront", description: Text("mobile_warehouse.ui.no_retailer_relationships"))
                } else {
                    ResponsiveGridContentWrapper {
                        ForEach(retailers) { retailer in
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
                    }
                }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.retailers")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.retailers()
                retailers = resp.retailers
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
