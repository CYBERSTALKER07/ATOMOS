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
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { Task { await loadDrivers() } }
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
                                Text("No trucks currently require a rescue.")
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(brokenDrivers) { driver in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(driver.name.isEmpty ? driver.driverId : driver.name)
                                                .font(.headline)
                                            Text("\(driver.vehicleLabel.isEmpty ? "—" : driver.vehicleLabel) · \(driver.truckStatus)")
                                                .font(.caption)
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        Button("Find Rescue") {
                                            Task { await findRescue(driver) }
                                        }
                                        .disabled(previewLoading)
                                        .buttonStyle(.borderedProminent)
                                    }
                                }
                            }
                        }
                        if let selectedBroken {
                            Section("Rescue Options for \(selectedBroken.name.isEmpty ? selectedBroken.driverId : selectedBroken.name)") {
                                if previewLoading {
                                    ProgressView()
                                } else if rescueOptions.isEmpty {
                                    Text("No rescuers available.")
                                        .foregroundStyle(.secondary)
                                } else {
                                    ForEach(rescueOptions) { opt in
                                        HStack {
                                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                                Text(opt.name.isEmpty ? opt.driverId : opt.name)
                                                    .font(.headline)
                                                Text("\(opt.licensePlate) · Capacity: \(String(format: "%.1f", opt.effectiveCapacityVu)) VU")
                                                    .font(.caption)
                                                    .foregroundStyle(.secondary)
                                                if opt.isCapacityExceeded {
                                                    Text("Insufficient capacity")
                                                        .font(.caption)
                                                        .foregroundStyle(.red)
                                                }
                                            }
                                            Spacer()
                                            Button("Propose") {
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
            .navigationTitle("Fleet Rescues")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { Task { await loadDrivers() } }
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
