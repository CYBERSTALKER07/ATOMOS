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
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if retailers.isEmpty {
                    ContentUnavailableView("No Retailers", systemImage: "storefront", description: Text("No retailer relationships"))
                } else {
                    List(retailers) { retailer in
                        HStack {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(retailer.name)
                                    .font(.headline)
                                Text(retailer.phone)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                                Text("\(retailer.totalOrders) orders")
                                    .font(.caption)
                                Text("\(retailer.totalRevenue.formatted()) UZS")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Retailers")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
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
