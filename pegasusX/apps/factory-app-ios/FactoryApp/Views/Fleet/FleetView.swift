import SwiftUI

struct FleetView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var vehicles: [Vehicle] = []
    @State private var liveRoutes: [FactoryFleetLiveRoute] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading && vehicles.isEmpty {
                    FactoryLoadingView(
                        title: "Loading fleet",
                        message: "Fetching registered vehicles and assignment status."
                    )
                } else if let error {
                    FactoryErrorView(message: error, retry: { load() })
                } else if vehicles.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No vehicles",
                        message: "No vehicles are registered for this factory yet."
                    )
                } else {
                    ResponsiveGridContentWrapper {
                        if !liveRoutes.isEmpty {
                            Section {
                                FactorySectionHeader(
                                    title: "Live drivers",
                                    subtitle: "\(liveRoutes.count) sealed/dispatched with GPS"
                                )
                                .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                                .listRowBackground(Color.clear)
                            }
                            Section {
                                ForEach(liveRoutes) { route in
                                    let lat = route.driverLocation?.lat ?? route.driverLocation?.latitude
                                    let lng = route.driverLocation?.lng ?? route.driverLocation?.longitude
                                    HStack {
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(route.driverName?.isEmpty == false ? route.driverName! : (route.driverId ?? route.manifestId))
                                                .font(.subheadline.bold())
                                            Text(
                                                lat != nil && lng != nil
                                                    ? String(format: "%.5f, %.5f", lat!, lng!)
                                                    : "Waiting for GPS"
                                            )
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        FactoryStatusBadge(text: (route.liveLocationAvailable ?? false) ? "LIVE" : "OFFLINE")
                                    }
                                }
                            }
                        }
                        Section {
                            FactorySectionHeader(
                                title: "Fleet roster",
                                subtitle: "\(vehicles.count) vehicles on record"
                            )
                            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                            .listRowBackground(Color.clear)
                        }

                        Section {
                            ForEach(Array(vehicles.enumerated()), id: \.element.id) { index, vehicle in
                                HStack(spacing: LabTheme.spacingLG) {
                                    Image(systemName: "truck.box")
                                        .font(.title2)
                                        .foregroundStyle(.secondary)
                                        .frame(width: 32)
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(vehicle.plateNumber)
                                            .font(.subheadline.bold())
                                        Text(vehicle.driverName.isEmpty ? "Unassigned" : vehicle.driverName)
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                        Text(L10n.format("mobile_factory.ui.capacitykgkg_capacityll", "\(Int(vehicle.capacityKg))", "\(Int(vehicle.capacityL))"))
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    FactoryStatusBadge(text: vehicle.status)
                                }
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
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise", action: { load() })
                        .labelStyle(.iconOnly)
                }
            }
            .task { load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                        load(silent: true)
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent {
            loading = true
        }
        error = nil
        Task {
            do {
                vehicles = try await FactoryService.fleet().vehicles
                if let live = try? await FactoryService.fleetLiveMap() {
                    liveRoutes = live.routes
                }
            } catch {
                self.error = error.localizedDescription
            }
            if !silent {
                loading = false
            }
        }
    }
}
