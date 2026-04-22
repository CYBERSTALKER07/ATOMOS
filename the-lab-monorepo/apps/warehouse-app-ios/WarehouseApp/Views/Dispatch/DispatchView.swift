import SwiftUI

struct DispatchView: View {
    @State private var preview: DispatchPreview?
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0

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
                } else if let preview {
                    VStack(spacing: 0) {
                        Picker("View", selection: $selectedSegment) {
                            Text("Orders (\(preview.undispatchedOrders.count))").tag(0)
                            Text("Drivers (\(preview.availableDrivers.count))").tag(1)
                        }
                        .pickerStyle(.segmented)
                        .padding()

                        if selectedSegment == 0 {
                            if preview.undispatchedOrders.isEmpty {
                                ContentUnavailableView("All Dispatched", systemImage: "checkmark.circle", description: Text("No pending orders"))
                            } else {
                                List(preview.undispatchedOrders) { order in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(order.retailerName.isEmpty ? String(order.orderId.prefix(8)) : order.retailerName)
                                                .font(.headline)
                                            Text("\(order.totalUzs.formatted()) UZS · \(order.itemCount) items")
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                }
                                .listStyle(.insetGrouped)
                            }
                        } else {
                            if preview.availableDrivers.isEmpty {
                                ContentUnavailableView("No Drivers", systemImage: "person.badge.key", description: Text("No available drivers"))
                            } else {
                                List(preview.availableDrivers) { driver in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(driver.name)
                                                .font(.headline)
                                            Text(driver.vehicleLabel.isEmpty ? "No vehicle" : driver.vehicleLabel)
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                }
                                .listStyle(.insetGrouped)
                            }
                        }
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Dispatch")
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
                preview = try await WarehouseService.dispatchPreview()
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
