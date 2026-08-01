import SwiftUI

struct LocationsView: View {
    @State private var items: [RetailerLocationDTO] = []
    @State private var activeId = ""
    @State private var loading = true
    @State private var banner: String?
    @State private var name = ""
    @State private var address = ""
    @State private var busy = false

    private let api = APIClient.shared

    var body: some View {
        List {
            Section {
                Text("Primary store is auto-created from your shop profile. Add branches for multi-location.")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Add branch") {
                TextField("Name", text: $name)
                TextField("Address", text: $address)
                Button(busy ? "…" : "Create") { Task { await create() } }
                    .disabled(busy)
            }
            Section("Branches") {
                if loading && items.isEmpty { ProgressView() }
                ForEach(items) { loc in
                    VStack(alignment: .leading, spacing: 6) {
                        HStack {
                            Text(loc.name).font(.headline)
                            if loc.isPrimary {
                                Text("Primary").font(.caption2).foregroundStyle(AppTheme.textTertiary)
                            }
                            if activeId == loc.locationId {
                                Text("Active").font(.caption2).foregroundStyle(AppTheme.accent)
                            }
                        }
                        Text(loc.deliveryAddress ?? "No address")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                        if loc.isActive && activeId != loc.locationId {
                            Button("Use for checkout") {
                                Task { await switchTo(loc.locationId) }
                            }
                            .font(.caption)
                        }
                        if loc.isActive && !loc.isPrimary {
                            Button("Set primary") {
                                Task { await setPrimary(loc.locationId) }
                            }
                            .font(.caption)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .navigationTitle("Locations")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            let res = try await api.getLocations()
            items = res.items
            activeId = res.activeLocationId ?? ""
        } catch {
            banner = error.localizedDescription
        }
    }

    private func create() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createLocation(name: name, address: address)
            name = ""; address = ""
            banner = "Created"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func switchTo(_ id: String) async {
        do {
            _ = try await api.switchLocation(locationId: id)
            activeId = id
            banner = "Switched active branch"
        } catch {
            banner = error.localizedDescription
        }
    }

    private func setPrimary(_ id: String) async {
        do {
            _ = try await api.setPrimaryLocation(locationId: id)
            banner = "Primary updated"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }
}
