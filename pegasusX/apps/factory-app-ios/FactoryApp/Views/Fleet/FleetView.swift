import SwiftUI

struct FleetView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var vehicles: [Vehicle] = []
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
                                        Text("\(Int(vehicle.capacityKg))kg · \(Int(vehicle.capacityL))L")
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
            .navigationTitle("Fleet")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise", action: { load() })
                        .labelStyle(.iconOnly)
                }
            }
            .task { load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard let eventType = event.eventType else { return }
                        guard eventType == .transferUpdate || eventType == .manifestUpdate else { return }
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
            } catch {
                self.error = error.localizedDescription
            }
            if !silent {
                loading = false
            }
        }
    }
}
