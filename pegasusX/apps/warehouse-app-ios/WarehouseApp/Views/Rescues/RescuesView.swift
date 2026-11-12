import SwiftUI

struct RescuesView: View {
    @State private var vm = RescuesViewModel()

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
                        Task { await vm.loadDrivers() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else {
                List {
                    Section("Needs Rescue") {
                        if let statusMessage = vm.statusMessage {
                            Text(statusMessage)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if vm.brokenDrivers.isEmpty {
                            Text("warehouse_portal.residual.text.no_trucks_currently_require_a_rescue")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(vm.brokenDrivers) { driver in
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
                                        PegasusHaptics.light()
                                        Task { await vm.findRescue(driver) }
                                    }
                                    .frame(minHeight: 44)
                                    .contentShape(Rectangle())
                                    .disabled(vm.previewLoading)
                                    .buttonStyle(.borderedProminent)
                                }
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("Driver \(driver.name.isEmpty ? driver.driverId : driver.name), truck status \(driver.truckStatus)")
                            }
                        }
                    }
                    if let selectedBroken = vm.selectedBroken {
                        Section(L10n.format("mobile_warehouse.ui.rescue_options_for_name", "\(selectedBroken.name.isEmpty ? selectedBroken.driverId : selectedBroken.name)")) {
                            if vm.previewLoading {
                                ProgressView()
                            } else if vm.rescueOptions.isEmpty {
                                Text("mobile_warehouse.ui.no_rescuers_available")
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(vm.rescueOptions) { opt in
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
                                            PegasusHaptics.light()
                                            Task { await vm.propose(opt) }
                                        }
                                        .frame(minHeight: 44)
                                        .contentShape(Rectangle())
                                        .disabled(opt.isCapacityExceeded || vm.proposeLoading)
                                        .buttonStyle(.bordered)
                                    }
                                    .accessibilityElement(children: .combine)
                                    .accessibilityLabel("Rescuer \(opt.name.isEmpty ? opt.driverId : opt.name), plate \(opt.licensePlate), capacity \(String(format: "%.1f", opt.effectiveCapacityVu)) VU")
                                }
                            }
                        }
                    }
                }
                .listStyle(.insetGrouped)
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("warehouse_portal.dispatch.rescues.text.fleet_rescues")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.loadDrivers() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh fleet rescues")
            }
        }
        .task { await vm.loadDrivers() }
        .refreshable { await vm.loadDrivers() }
    }
}
