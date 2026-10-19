import SwiftUI

struct RescuesView: View {
    @State private var brokenDrivers: [AvailableDriver] = []
    @State private var selectedBroken: AvailableDriver?
    @State private var rescueOptions: [RescueOption] = []
    @State private var loading = true
    @State private var previewLoading = false
    @State private var proposeLoading = false
    @State private var error: String?
    @State private var statusMessage: String?

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
                        Button("common.action.retry") { Task { await loadDrivers() } }
                    }
                } else {
                    List {
                        Section("Needs Rescue") {
                            if let statusMessage {
                                Text(statusMessage)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            if brokenDrivers.isEmpty {
                                Text("warehouse_portal.residual.text.no_trucks_currently_require_a_rescue")
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(brokenDrivers) { driver in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(driver.name.isEmpty ? driver.driverId : driver.name)
                                                .font(.headline)
                                            Text(L10n.format("mobile_warehouse.ui.vehiclelabel_truckstatus", "\(driver.vehicleLabel.isEmpty ? "—" : driver.vehicleLabel)", "\(driver.truckStatus)"))
                                                .font(.caption)
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        Button("mobile_warehouse.ui.find_rescue") {
                                            Task { await findRescue(driver) }
                                        }
                                        .disabled(previewLoading)
                                        .buttonStyle(.borderedProminent)
                                    }
                                }
                            }
                        }
                        if let selectedBroken {
                            Section(L10n.format("mobile_warehouse.ui.rescue_options_for_name", "\(selectedBroken.name.isEmpty ? selectedBroken.driverId : selectedBroken.name)")) {
                                if previewLoading {
                                    ProgressView()
                                } else if rescueOptions.isEmpty {
                                    Text("mobile_warehouse.ui.no_rescuers_available")
                                        .foregroundStyle(.secondary)
                                } else {
                                    ForEach(rescueOptions) { opt in
                                        HStack {
                                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                                Text(opt.name.isEmpty ? opt.driverId : opt.name)
                                                    .font(.headline)
                                                Text(L10n.format("mobile_warehouse.ui.licenseplate_capacity_effectivecapacityvu_vu", "\(opt.licensePlate)", "\(String(format: "%.1f", opt.effectiveCapacityVu))"))
                                                    .font(.caption)
                                                    .foregroundStyle(.secondary)
                                                if opt.isCapacityExceeded {
                                                    Text("mobile_warehouse.ui.insufficient_capacity")
                                                        .font(.caption)
                                                        .foregroundStyle(.red)
                                                }
                                            }
                                            Spacer()
                                            Button("mobile_warehouse.ui.propose") {
                                                Task { await propose(opt) }
                                            }
                                            .disabled(opt.isCapacityExceeded || proposeLoading)
                                            .buttonStyle(.bordered)
                                        }
                                    }
                                }
                            }
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("warehouse_portal.dispatch.rescues.text.fleet_rescues")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { Task { await loadDrivers() } }
                }
            }
            .task { await loadDrivers() }
            .refreshable { await loadDrivers() }
        }
    }

    private func loadDrivers() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let preview = try await WarehouseService.dispatchPreview()
            brokenDrivers = (preview.availableDrivers + preview.unavailableDrivers)
                .filter { $0.truckStatus.uppercased() == "NEEDS_RESCUE" }
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func findRescue(_ driver: AvailableDriver) async {
        selectedBroken = driver
        rescueOptions = []
        previewLoading = true
        statusMessage = nil
        defer { previewLoading = false }
        do {
            let resp = try await WarehouseService.previewRescue(brokenDriverId: driver.driverId)
            rescueOptions = resp.rescueOptions
        } catch {
            statusMessage = error.localizedDescription
            selectedBroken = nil
        }
    }

    private func propose(_ option: RescueOption) async {
        guard let broken = selectedBroken else { return }
        proposeLoading = true
        defer { proposeLoading = false }
        do {
            let rescueId = UUID().uuidString
            _ = try await WarehouseService.proposeRescue(
                rescueId: rescueId,
                brokenDriverId: broken.driverId,
                rescueDriverId: option.driverId
            )
            statusMessage = "Rescue proposed"
            selectedBroken = nil
            rescueOptions = []
            await loadDrivers()
        } catch {
            statusMessage = error.localizedDescription
        }
    }
}
