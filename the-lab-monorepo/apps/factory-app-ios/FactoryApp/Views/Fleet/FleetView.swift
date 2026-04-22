import SwiftUI

struct FleetView: View {
    @State private var vehicles: [Vehicle] = []
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
                } else if vehicles.isEmpty {
                    ContentUnavailableView("No Vehicles", systemImage: "truck.box", description: Text("No vehicles registered"))
                } else {
                    List {
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
                                Text(vehicle.status)
                                    .font(.caption2.bold())
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 3)
                                    .background(.quaternary)
                                    .clipShape(Capsule())
                            }
                            .staggeredAppear(index: index)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Fleet")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .task { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                vehicles = try await FactoryService.fleet().vehicles
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
