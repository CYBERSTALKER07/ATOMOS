import SwiftUI

struct FleetView: View {
    @State private var vm = FleetViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.vehicles.isEmpty {
                FactoryLoadingView(
                    title: "Loading fleet",
                    message: "Fetching registered vehicles and assignment status."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else if vm.vehicles.isEmpty {
                FactoryStateView(
                    kind: .empty,
                    headline: "No vehicles",
                    message: "No vehicles are registered for this factory yet."
                )
            } else {
                ResponsiveGridContentWrapper {
                    if !vm.liveRoutes.isEmpty {
                        Section {
                            FactorySectionHeader(
                                title: "Live drivers",
                                subtitle: "\(vm.liveRoutes.count) sealed/dispatched with GPS"
                            )
                            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                            .listRowBackground(Color.clear)
                        }
                        Section {
                            ForEach(vm.liveRoutes) { route in
                                let lat = route.driverLocation?.lat ?? route.driverLocation?.latitude
                                let lng = route.driverLocation?.lng ?? route.driverLocation?.longitude
                                let driverLabel = resolvedDriverLabel(for: route)
                                let coordText = resolvedCoordText(lat: lat, lng: lng)
                                let liveBadge = (route.liveLocationAvailable ?? false) ? "LIVE" : "OFFLINE"

                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(driverLabel)
                                            .font(.subheadline.bold())
                                        Text(coordText)
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    FactoryStatusBadge(text: liveBadge)
                                }
                                .frame(minHeight: 44)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("Driver \(driverLabel), \(liveBadge), \(coordText)")
                            }
                        }
                    }

                    Section {
                        FactorySectionHeader(
                            title: "Fleet roster",
                            subtitle: "\(vm.vehicles.count) vehicles on record"
                        )
                        .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                        .listRowBackground(Color.clear)
                    }

                    Section {
                        ForEach(Array(vm.vehicles.enumerated()), id: \.element.id) { index, vehicle in
                            let assignedDriver = vehicle.driverName.isEmpty ? "Unassigned" : vehicle.driverName
                            HStack(spacing: LabTheme.spacingLG) {
                                Image(systemName: "truck.box")
                                    .font(.title2)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 32)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(vehicle.plateNumber)
                                        .font(.subheadline.bold())
                                    Text(assignedDriver)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Text(L10n.format("mobile_factory.ui.capacitykgkg_capacityll", "\(Int(vehicle.capacityKg))", "\(Int(vehicle.capacityL))"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                FactoryStatusBadge(text: vehicle.status)
                            }
                            .frame(minHeight: 44)
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Vehicle \(vehicle.plateNumber), driver \(assignedDriver), status \(vehicle.status)")
                            .staggeredAppear(index: index)
                        }
                    }
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.fleet")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh fleet")
            }
        }
        .task {
            vm.startRealtime()
            await vm.load()
        }
        .onDisappear {
            vm.stopRealtime()
        }
    }

    private func resolvedDriverLabel(for route: FactoryFleetLiveRoute) -> String {
        if let name = route.driverName, !name.isEmpty {
            return name
        }
        return route.driverId ?? route.manifestId
    }

    private func resolvedCoordText(lat: Double?, lng: Double?) -> String {
        guard let lat, let lng else {
            return "Waiting for GPS"
        }
        return String(format: "%.5f, %.5f", lat, lng)
    }
}
